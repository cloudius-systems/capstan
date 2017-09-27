/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 * Modifications copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mikelangelo-project/capstan/core"
	"github.com/mikelangelo-project/capstan/image"
	"gopkg.in/yaml.v2"
)

const (
	DefaultRepositoryUrl = "https://mikelangelo-capstan.s3.amazonaws.com/"
)

type Repo struct {
	URL        string
	Path       string
	DisableKvm bool
}

type CapstanSettings struct {
	RepoUrl    string `yaml:"repo_url"`
	DisableKvm bool   `yaml:"disable_kvm"`
}

func NewRepo(url string) *Repo {
	root := os.Getenv("CAPSTAN_ROOT")
	if root == "" {
		root = filepath.Join(HomePath(), "/.capstan/")
	}

	// Read configuration file
	config := CapstanSettings{
		RepoUrl:    "",
		DisableKvm: false,
	}
	data, err := ioutil.ReadFile(filepath.Join(root, "config.yaml"))
	if err == nil {
		err = yaml.Unmarshal(data, &config)
	}

	// Decide which repo URL to choose. Take first non-empty value of:
	// 1. -u
	// 2. Capstan.yaml, if contains CAPSTAN_REPO_URL
	// 3. Env variable CAPSTAN_REPO_URL
	// 4. Default
	// Config file preceeds Env variable to enable per-capstan-root config.
	url = func(flagUrl string) string {
		if flagUrl != "" {
			return flagUrl
		}
		if config.RepoUrl != "" {
			return config.RepoUrl
		}
		if envUrl := os.Getenv("CAPSTAN_REPO_URL"); envUrl != "" {
			return envUrl
		}
		return DefaultRepositoryUrl
	}(url)

	// Attempt to load DisableKvm flag from environment.
	if envDisableKvm, err := strconv.ParseBool(os.Getenv("CAPSTAN_DISABLE_KVM")); err == nil {
		config.DisableKvm = envDisableKvm
	}

	return &Repo{
		URL:        url,
		Path:       root,
		DisableKvm: config.DisableKvm,
	}
}

type ImageInfo struct {
	FormatVersion string `yaml:"format_version"`
	Version       string
	Created       string
	Description   string
	Build         string
}

func (r *Repo) PrintRepo() {
	fmt.Printf("CAPSTAN_ROOT: %s\n", r.Path)
	fmt.Printf("CAPSTAN_REPO_URL: %s\n", r.URL)
	fmt.Printf("CAPSTAN_DISABLE_KVM: %v\n", r.DisableKvm)
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
	fmt.Printf("Importing into %s\n", dst)
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

// PackageExists will check that both package manifest and package file are
// present in the local package repository.
func (r *Repo) PackageExists(packageName string) bool {
	if _, err := os.Stat(r.PackageManifest(packageName)); os.IsNotExist(err) {
		return false
	}

	if _, err := os.Stat(r.PackagePath(packageName)); os.IsNotExist(err) {
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

func (r *Repo) PackagesPath() string {
	return filepath.Join(r.Path, "packages")
}

func (r *Repo) ImagePath(hypervisor string, image string) string {
	return filepath.Join(r.RepoPath(), image, fmt.Sprintf("%s.%s", filepath.Base(image), hypervisor))
}

func (r *Repo) ImageCachePath(hypervisor string, image string) string {
	return filepath.Join(r.RepoPath(), image, fmt.Sprintf("%s.%s.cache", filepath.Base(image), hypervisor))
}

func (r *Repo) PackagePath(packageName string) string {
	return filepath.Join(r.Path, "packages", fmt.Sprintf("%s.mpm", packageName))
}

func (r *Repo) PackageManifest(packageName string) string {
	return filepath.Join(r.Path, "packages", fmt.Sprintf("%s.yaml", packageName))
}

func (r *Repo) ListImages() string {
	res := fmt.Sprintln(FileInfoHeader())
	namespaces, _ := ioutil.ReadDir(r.RepoPath())
	for _, n := range namespaces {
		images, _ := ioutil.ReadDir(filepath.Join(r.RepoPath(), n.Name()))
		for _, i := range images {
			namespace := ""
			directory := n.Name()

			if i.IsDir() {
				namespace = n.Name()
				directory = i.Name()
			} else if !strings.HasSuffix(i.Name(), ".qemu") {
				continue
			}
			info, err := ParseIndexYaml(r.RepoPath(), namespace, directory)
			if err != nil {
				fmt.Println(err)
				info = &FileInfo{Name: directory, Namespace: namespace}
			}
			res += fmt.Sprintln(info.String())
		}
	}
	return res
}

func (r *Repo) ListPackages() string {
	res := fmt.Sprintln(FileInfoHeader())
	packages, _ := ioutil.ReadDir(r.PackagesPath())
	for _, p := range packages {
		if filepath.Ext(p.Name()) == ".yaml" {
			pkg, err := core.ParsePackageManifest(filepath.Join(r.PackagesPath(), p.Name()))

			// Skip the package if the manifest cannot be parsed.
			if err != nil {
				continue
			}

			res += fmt.Sprintln(pkg.String())
		}
	}
	return res
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
	// Temporarily use the mike/osv-loader image. Note that in order for this to work
	// one has to actually import mike/osv-loader image first!
	//
	// capstan import mike/osv-loader /path/to/osv/build/release/loader.img
	if loaderImage == "" {
		loaderImage = "mike/osv-loader"
	}

	// Get the actual path of the loader image.
	loaderImagePath := r.ImagePath("qemu", loaderImage)
	// Check whether the base loader image exists
	loaderInfo, err := os.Stat(loaderImagePath)
	if os.IsNotExist(err) {
		fmt.Printf("The specified loader image (%s) does not exist.\n", loaderImagePath)
		return err
	}

	// Get the size of the loader image, then round that to the closest 2MB to start the user
	// ZFS partition.
	zfsStart := (loaderInfo.Size() + 2097151) & ^2097151
	// Make filesystem size in bytes
	zfsSize := int64(imageSize * 1024 * 1024)
	// Adjust user partition size so that total image size will be as defined by user.
	zfsSize -= zfsStart

	if zfsSize <= 0 {
		return fmt.Errorf("Image size (%d B) not sufficient for loader image content (%d B)",
			int64(imageSize*1024*1024), zfsStart)
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

	// Convert the image to QCOW2 format. This will prevent the image file from
	// becoming to large in the next step when we actually resize it.
	if err := ConvertImageToQCOW2(imagePath); err != nil {
		return err
	}

	// Store the information about the partition into the image.
	if err := SetPartition(imagePath, 2, uint64(zfsStart), uint64(zfsSize)); err != nil {
		fmt.Printf("Setting the ZFS partition failed for %s\n", imagePath)
		return err
	}

	// Now that the partition has been created, resize the virtual image size.
	if err := ResizeImage(imagePath, uint64(zfsSize+zfsStart)); err != nil {
		fmt.Printf("Failed to set the target size (%db) of the image %s\n", (zfsSize + zfsStart), imagePath)
		return err
	}

	// The image can now be imported into Capstan's repository.
	return r.ImportImage(imageName, imagePath, "", time.Now().Format(core.FRIENDLY_TIME_F), "", "")
}

func (r *Repo) ImportPackage(pkg core.Package, packagePath string) error {
	fmt.Printf("Importing package %s...\n", packagePath)

	// Get the root of the packages dir.
	dir := filepath.Join(r.Path, "packages")

	// Make sure the path exists by creating the entire directory structure.
	err := os.MkdirAll(dir, 0775)
	if err != nil {
		return fmt.Errorf("%s: mkdir failed", dir)
	}

	// Get the filename of the package...
	packageFileName := filepath.Base(packagePath)
	// ... and prepare the target file name.
	target := filepath.Join(dir, packageFileName)

	// Copy the package into the repository.
	err = CopyLocalFile(target, packagePath)
	if err != nil {
		fmt.Printf("Failed to import package into %s\n", packagePath)
		return err
	}

	// Store package metadata descriptor into the repository.
	d, err := yaml.Marshal(pkg)
	if err != nil {
		// Since there was en error exporting YAML file, remove the package file.
		os.Remove(target)

		return err
	}

	manifestFile := strings.TrimSuffix(packageFileName, filepath.Ext(packageFileName))
	err = ioutil.WriteFile(filepath.Join(dir, fmt.Sprintf("%s.yaml", manifestFile)), d, 0644)
	if err != nil {
		// Since there was en error exporting YAML file, remove the package file.
		os.Remove(target)

		return err
	}

	fmt.Printf("Package %s successfully imported into repository %s\n", packageFileName, dir)
	return nil
}

func (r *Repo) GetPackage(pkgname string) (io.ReadSeeker, error) {
	pkgpath := r.PackagePath(pkgname)

	// Make sure the package does exist.
	if _, err := os.Stat(pkgpath); os.IsNotExist(err) {
		return nil, err
	}

	return os.Open(pkgpath)
}

// GetPackageTarReader returns tar reader for package with given name.
func (r *Repo) GetPackageTarReader(pkgname string) (*tar.Reader, error) {
	reader, err := r.GetPackage(pkgname)
	if err != nil {
		return nil, err
	}

	// Load package (tar.gz or tar supported).
	if gzReader, err := gzip.NewReader(reader); err == nil {
		return tar.NewReader(gzReader), nil
	} else if err == gzip.ErrHeader {
		reader.Seek(0, io.SeekStart) // revert offset that gzReader has corrupted
		return tar.NewReader(reader), nil
	} else {
		return nil, err
	}
}

func (r *Repo) GetPackageDependencies(pkg core.Package, downloadMissing bool) ([]core.Package, error) {
	var dependencies []core.Package

	for _, requiredPackage := range pkg.Require {
		// If the package does not exist in the local repository and the request
		// was made to download missing packages we should try to download them
		// from the remote repository.
		if !r.PackageExists(requiredPackage) {
			if downloadMissing {
				if err := r.DownloadPackage(r.URL, requiredPackage); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("Package %s does not exist in your local repository. Pull it manually using "+
					"'capstan package pull %s' or enable automatic pulling of missing "+
					"packages by adding --pull-missing flag", requiredPackage, requiredPackage)
			}
		}

		// Proceed with the evaluation of the package content.
		rpkg, err := core.ParsePackageManifest(r.PackageManifest(requiredPackage))
		if err != nil {
			return nil, err
		}

		// Process all additional required packages.
		rdeps, err := r.GetPackageDependencies(rpkg, downloadMissing)
		if err != nil {
			return nil, err
		}

		dependencies = append(dependencies, rpkg)
		dependencies = append(dependencies, rdeps...)
	}

	return dependencies, nil
}

func mergeDependencies(existing []core.Package, additional []core.Package) []core.Package {
	for _, newpkg := range additional {
		// Check if the package has already been added as a dependency.
		exists := false
		for _, existingpkg := range existing {
			if existingpkg.Name == newpkg.Name {
				exists = true
				break
			}
		}

		if !exists {
			existing = append(existing, newpkg)
		}
	}

	return existing
}
