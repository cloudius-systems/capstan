package core

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type Package struct {
	Name    string
	Title   string
	Author  string            "author,omitempty"
	Version string            "version,omitempty"
	Require []string          "require,omitempty"
	Binary  map[string]string "binary,omitempty"
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
		return pkg, err
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
