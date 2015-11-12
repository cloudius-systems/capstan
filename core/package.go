package core

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

type Package struct {
	Name    string
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

	if p.Author == "" {
		return fmt.Errorf("'author' must be provided for the package")
	}

	return nil
}
