package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan/core"
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

func ComposePackage(packageDir string) error {
	packageDir, err := filepath.Abs(packageDir)

	// First, look for the package metadata.
	pkgMetadata := filepath.Join(packageDir, "meta", "package.yaml")

	if _, err := os.Stat(pkgMetadata); os.IsNotExist(err) {
		return fmt.Errorf("%s does not seem to be a package (missing meta/package.yaml file)", packageDir)
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

	// If all is well, we have to start preparing the files for upload.
	paths := make(map[string]string)
	if err := CollectPackageContents(paths, packageDir); err != nil {
		return err
	}

	return nil
}

func CollectPackageContents(contents map[string]string, packageDir string) error {
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist", packageDir)
	}

	err := filepath.Walk(packageDir, func(path string, info os.FileInfo, _ error) error {
		relPath := strings.TrimPrefix(path, packageDir)
		// Ignore package's meta data
		if relPath != "" && !strings.HasPrefix(relPath, "/meta") {
			contents[path] = relPath
		}
		return nil
	})

	return err
}
