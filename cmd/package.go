package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan/core"
	"github.com/cloudius-systems/capstan/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func InitPackage(packageName string, p *core.Package) error {
	// We have to create hte package directory and it's metadata directory.
	metaPath := filepath.Join(packageName, "meta")

	fmt.Printf("Initializing package in %s\n", metaPath)

	// Create the meta dir.
	if err := os.MkdirAll(metaPath, 0755); err != nil {
		return err
	}

	// Save basic package data to YAML.
	d, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	// Save serialised YAML file to the appropriate place in the metadata directory.
	err = ioutil.WriteFile(filepath.Join(metaPath, "package.yaml"), d, 0644)
	if err != nil {
		return err
	}

	return nil
}

func ComposePackage(repo *util.Repo, imageSize int64, packageDir string, appName string) error {
	// If all is well, we have to start preparing the files for upload.
	paths := make(map[string]string)

	// We have to include the "bootstrap" package
	bootstrap := repo.PackagePath("bootstrap")
	if err := CollectDirectoryContents(paths, bootstrap, repo); err != nil {
		return err
	}

	if err := CollectDirectoryContents(paths, packageDir, repo); err != nil {
		return err
	}

	// Initialize an empty image based on the provided loader image. imageSize is used to
	// determine the size of the user partition. Use default loader image.
	if err := repo.InitializeImage("", appName, imageSize); err != nil {
		return fmt.Errorf("Failed to initialize empty image named %s", appName)
	}

	// Get the path of imported image.
	imagePath := repo.ImagePath("qemu", appName)

	// Upload the specified path onto virtual image.
	if err := UploadPackageContents(imagePath, paths); err != nil {
		return err
	}

	return nil
}

func CollectDirectoryContents(contents map[string]string, packageDir string, repo *util.Repo) error {
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist", packageDir)
	}

	packageDir, err := filepath.Abs(packageDir)

	// First, look for the package metadata.
	pkgMetadata := filepath.Join(packageDir, "meta", "package.yaml")

	if _, err := os.Stat(pkgMetadata); os.IsNotExist(err) {
		return fmt.Errorf("%s is missing package description in meta/package.yaml", packageDir)
	}

	// If the file exists, try to parse it.
	d, err := ioutil.ReadFile(pkgMetadata)
	if err != nil {
		return err
	}

	// Now parse the package descriptior.
	var pkg core.Package
	if err := pkg.Parse(d); err != nil {
		return err
	}

	for _, requiredPackage := range pkg.Require {
		requiredPath := repo.PackagePath(requiredPackage)

		CollectDirectoryContents(contents, requiredPath, repo)
	}

	err = filepath.Walk(packageDir, func(path string, info os.FileInfo, _ error) error {
		relPath := strings.TrimPrefix(path, packageDir)
		// Ignore package's meta data
		if relPath != "" && !strings.HasPrefix(relPath, "/meta") {
			contents[path] = relPath
		}
		return nil
	})

	return err
}
