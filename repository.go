/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package capstan

import (
	"errors"
	"fmt"
	"github.com/cloudius-systems/capstan/image"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

type Repo struct {
	Path string
}

func NewRepo() *Repo {
	root := os.Getenv("CAPSTAN_ROOT")
	if root == "" {
		root = filepath.Join(os.Getenv("HOME"), "/.capstan/repository/") 
	}
	return &Repo{
		Path: root,
	}
}

func (r *Repo) PullImage(image string) error {
	workTree := r.workTree(image)
	if _, err := os.Stat(workTree); os.IsExist(err) {
		return r.updateImage(image)
	}
	fmt.Printf("Pulling %s...\n", image)
	gitUrl := fmt.Sprintf("https://github.com/%s", image)
	cmd := exec.Command("git", "clone", "--depth", "1", gitUrl, workTree)
	_, err := cmd.Output()
	if err != nil {
		return errors.New(fmt.Sprintf("%s: unable to pull remote image", image))
	}
	return nil
}

func (r *Repo) updateImage(image string) error {
	fmt.Printf("Updating %s...\n", image)
	workTree := r.workTree(image)
	gitDir   := r.gitDir(image)
	cmd := exec.Command("git", "--git-dir", gitDir, "--work-tree", workTree, "remote", "update")
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	cmd = exec.Command("git", "--git-dir", gitDir, "--work-tree", workTree, "merge", "origin/master")
	_, err = cmd.Output()
	return err
}

func (r *Repo) gitDir(image string) string {
	return filepath.Join(r.workTree(image), ".git")
}

func (r *Repo) workTree(image string) string {
	return filepath.Join(r.Path, image)
}

func (r *Repo) PushImage(imageName string, file string) error {
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
	case image.VMDK:
		hypervisor = "vmware"
	default:
		return fmt.Errorf("%s: unsupported image format", file)
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("%s: no such file", file))
	}
	fmt.Printf("Pushing %s...\n", imageName)
	cmd := exec.Command("mkdir", "-p", filepath.Dir(r.ImagePath(hypervisor, imageName)))
	_, err = cmd.Output()
	if err != nil {
		return errors.New(fmt.Sprintf("%s: mkdir failed", filepath.Dir(r.ImagePath(hypervisor, imageName))))
	}
	cmd = exec.Command("cp", file, r.ImagePath(hypervisor, imageName))
	_, err = cmd.Output()
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
	path := filepath.Join(r.Path, image)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("%s: no such image\n", image))
	}
	fmt.Printf("Removing %s...\n", image)
	cmd := exec.Command("rm", "-rf", path)
	_, err := cmd.Output()
	return err;
}

func (r *Repo) ImagePath(hypervisor string, image string) string {
	return filepath.Join(r.Path, image, fmt.Sprintf("%s.%s", filepath.Base(image), hypervisor))
}

func (r *Repo) ListImages() {
	namespaces, _ := ioutil.ReadDir(r.Path)
	for _, n := range namespaces {
		images, _ := ioutil.ReadDir(filepath.Join(r.Path, n.Name()))
		nrImages := 0
		for _, i := range images {
			if i.IsDir() {
				fmt.Println(n.Name() + "/" + i.Name())
				nrImages++
			}
		}
		// Image is directly at repository root with no namespace:
		if nrImages == 0 && n.IsDir() {
			fmt.Println(n.Name())
		}
	}
}

func (r *Repo) DefaultImage() string {
	if !ConfigExists("Capstanfile") {
		return ""
	}
	pwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	image := path.Base(pwd)
	return image
}
