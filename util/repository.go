/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"errors"
	"fmt"
	"github.com/cloudius-systems/capstan/core"
	"github.com/cloudius-systems/capstan/image"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	DefaultRepositoryUrl = "https://s3.amazonaws.com/osv.capstan/"
)

type Repo struct {
	URL  string
	Path string
}

func NewRepo(url string) *Repo {
	root := os.Getenv("CAPSTAN_ROOT")
	if root == "" {
		root = filepath.Join(HomePath(), "/.capstan/")
	}
	return &Repo{
		URL:  url,
		Path: root,
	}
}

type ImageInfo struct {
	FormatVersion string `yaml:"format_version"`
	Version       string
	Created       string
	Description   string
	Build         string
}

func (r *Repo) ImportImage(imageName string, file string, version string, created string, description string, build string) error {
	format, err := image.Probe(file)
	if err != nil {
		return err
	}
	var hypervisor string
	switch format {
	case image.VDI:
		hypervisor = "vbox"
	case image.QCOW2:
		hypervisor = "qemu"
	case image.RAW:
		hypervisor = "raw"
	case image.VMDK:
		hypervisor = "vmware"
	default:
		return fmt.Errorf("%s: unsupported image format", file)
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("%s: no such file", file))
	}
	fmt.Printf("Importing %s...\n", imageName)
	dir := filepath.Dir(r.ImagePath(hypervisor, imageName))
	err = os.MkdirAll(dir, 0775)
	if err != nil {
		return errors.New(fmt.Sprintf("%s: mkdir failed", dir))
	}

	dst := r.ImagePath(hypervisor, imageName)
	fmt.Printf("Importing into %s", dst)
	cmd := CopyFile(file, dst)
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	info := ImageInfo{
		FormatVersion: "1",
		Version:       version,
		Created:       created,
		Description:   description,
		Build:         build,
	}
	value, err := yaml.Marshal(info)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(dir, "index.yaml"), value, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repo) ImageExists(hypervisor, image string) bool {
	file := r.ImagePath(hypervisor, image)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

func (r *Repo) RemoveImage(image string) error {
	path := filepath.Join(r.RepoPath(), image)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("%s: no such image\n", image))
	}
	fmt.Printf("Removing %s...\n", image)
	err := os.RemoveAll(path)
	return err
}

func (r *Repo) RepoPath() string {
	return filepath.Join(r.Path, "repository")
}

func (r *Repo) ImagePath(hypervisor string, image string) string {
	return filepath.Join(r.RepoPath(), image, fmt.Sprintf("%s.%s", filepath.Base(image), hypervisor))
}

func (r *Repo) PackagePath(packageName string) string {
	return filepath.Join(r.Path, "packages", packageName)
}

func (r *Repo) ListImages() {
	fmt.Println(FileInfoHeader())
	namespaces, _ := ioutil.ReadDir(r.RepoPath())
	for _, n := range namespaces {
		images, _ := ioutil.ReadDir(filepath.Join(r.RepoPath(), n.Name()))
		nrImages := 0
		nrFiles := 0
		for _, i := range images {
			if i.IsDir() {
				info := MakeFileInfo(r.RepoPath(), n.Name(), i.Name())
				if info == nil {
					fmt.Println(n.Name() + "/" + i.Name())
				} else {
					fmt.Println(info.String())
				}
				nrImages++
			} else {
				nrFiles++
			}
		}
		// Image is directly at repository root with no namespace:
		if nrImages == 0 && nrFiles != 0 {
			fmt.Println(n.Name())
		}
	}
}

func (r *Repo) DefaultImage() string {
	if !core.IsTemplateFile("Capstanfile") {
		return ""
	}
	pwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	image := path.Base(pwd)
	return image
}

func (r *Repo) InitializeImage(loaderImage string, imageName string, imageSize int64) error {
	// Temporarily use the mike/osv-launcher image. Note that in order for this to work
	// one has to actually import mike/osv-launcher image first!
	//
	// capstan import mike/osv-launcher /path/to/osv/build/release/loader.img
	if loaderImage == "" {
		loaderImage = "mike/osv-launcher"
	}

	// Get the actual path of the launcher image.
	loaderImagePath := r.ImagePath("raw", loaderImage)
	// Check whether the base launcher image exists
	loaderInfo, err := os.Stat(loaderImagePath)
	if os.IsNotExist(err) {
		fmt.Printf("The specified loader image (%s) does not exist.\n", loaderImagePath)
		return err
	}

	// Create temporary folder in which the image will be composed.
	tmp, _ := ioutil.TempDir("", "capstan")
	// Once this function is finished, remove temporary file.
	defer os.RemoveAll(tmp)
	imagePath := path.Join(tmp, "application.img")

	// Copy the OSv base iamge into application image
	if err := CopyLocalFile(imagePath, loaderImagePath); err != nil {
		return err
	}

	// Get the size of the loader image, then round that to the closest 2MB to start the user
	// ZFS partition.
	zfsStart := (loaderInfo.Size() + 2097151) & ^2097151
	// Make filesystem size in bytes
	zfsSize := int64(imageSize * 1024 * 1024)

	// Make sure the image is in QCOW2 format. This is to make sure that the
	// image in the next step does not grow in size in case the input image is
	// in RAW format.
	if err := SetPartition(imagePath, 2, uint64(zfsStart), uint64(zfsSize)); err != nil {
		fmt.Printf("Setting the ZFS partition failed for %s\n", imagePath)
		return err
	}

	// Convert the image to QCOW2 format. This will prevent the image file from
	// becoming to large in the next step when we actually resize it.
	if err := ConvertImageToQCOW2(imagePath); err != nil {
		return err
	}

	// Now that the partition has been created, resize the virtual image size.
	if err := ResizeImage(imagePath, uint64(zfsSize+zfsStart)); err != nil {
		fmt.Printf("Failed to set the target size (%db) of the image %s\n", (zfsSize + zfsStart), imagePath)
		return err
	}

	// The image can now be imported into Capstan's repository.
	return r.ImportImage(imageName, imagePath, "", time.Now().Format(time.RFC3339), "", "")
}
