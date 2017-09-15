/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Package struct {
	Name    string
	Title   string
	Author  string            `yaml:"author,omitempty"`
	Version string            `yaml:"version,omitempty"`
	Require []string          `yaml:"require,omitempty"`
	Binary  map[string]string `yaml:"binary,omitempty"`
	Created YamlTime          `yaml:"created"`
}

func (p *Package) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, p); err != nil {
		return err
	}

	if p.Name == "" {
		return fmt.Errorf("'name' must be provided for the package")
	}

	if p.Title == "" {
		return fmt.Errorf("'title' must be provided for the package")
	}

	if p.Author == "" {
		return fmt.Errorf("'author' must be provided for the package")
	}

	return nil
}

func ParsePackageManifest(manifestFile string) (Package, error) {
	var pkg Package

	// Make sure the metadata file exists.
	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		return pkg, fmt.Errorf("Manifest file %s does not exist", manifestFile)
	}

	// Read the package descriptor.
	d, err := ioutil.ReadFile(manifestFile)
	if err != nil {
		return pkg, err
	}

	// And parse it. This must succeed in order to be able to proceed.
	if err := pkg.Parse(d); err != nil {
		return pkg, err
	}

	return pkg, nil
}

func (p *Package) String() string {
	return fmt.Sprintf("%-50s %-50s %-25s %-15s", p.Name, p.Title, p.Version, p.Created)
}
