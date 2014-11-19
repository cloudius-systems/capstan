/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core

import (
	"fmt"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// A template is a configuration file that describes how to build a VM image.
// It is usually representeed as a `Capstanfile` file on disk.
type Template struct {
	Base    string
	RpmBase *RpmPackage "rpm-base"
	Cmdline string
	Build   string
	Files   map[string]string
	Rootfs  string
}

// IsTemplateFile returns true if filename points to a valid template file;
// otherwise returns false.
func IsTemplateFile(filename string) bool {
	_, err := ReadTemplateFile(filename)
	return err == nil
}

// ReadTemplateFile parses a Template from a file.
func ReadTemplateFile(filename string) (*Template, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	c, err := ParseTemplate(data)
	if err != nil {
		return nil, err
	}
	if err := c.substituteVars(); err != nil {
		return nil, err
	}
	return c, nil
}

// ParseTemplate parses a Template from a byte array.
func ParseTemplate(data []byte) (*Template, error) {
	t := Template{}
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if t.Cmdline == "" {
		return nil, fmt.Errorf("\"cmdline\" not found")
	}
	if t.Rootfs == "" {
		t.Rootfs = "ROOTFS"
	} else {
		if _, err := os.Stat(t.Rootfs); os.IsNotExist(err) {
			fmt.Printf("Capstanfile: rootfs: %s does not exist\n", t.Rootfs)
			return nil, err
		}
	}
	return &t, nil
}

func (t *Template) substituteVars() error {
	for target, source := range t.Files {
		t.Files[target] = strings.Replace(source, "&", filepath.Base(target), -1)
	}
	return nil
}
