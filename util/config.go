/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"fmt"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Base    string
	RpmBase *RpmPackage "rpm-base"
	Cmdline string
	Build   string
	Files   map[string]string
	Rootfs  string
}

func ConfigExists(filename string) bool {
	_, err := ReadConfig(filename)
	return err == nil
}

func ReadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	c, err := ParseConfig(data)
	if err != nil {
		return nil, err
	}
	if err := c.substituteVars(); err != nil {
		return nil, err
	}
	return c, nil
}

func ParseConfig(data []byte) (*Config, error) {
	c := Config{}
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if c.Cmdline == "" {
		return nil, fmt.Errorf("\"cmdline\" not found")
	}
	if c.Rootfs == "" {
		c.Rootfs = "ROOTFS"
	} else {
		if _, err := os.Stat(c.Rootfs); os.IsNotExist(err) {
			fmt.Printf("Capstanfile: rootfs: %s does not exist\n", c.Rootfs)
			return nil, err
		}
	}
	return &c, nil
}

func (c *Config) substituteVars() error {
	for target, source := range c.Files {
		c.Files[target] = strings.Replace(source, "&", filepath.Base(target), -1)
	}
	return nil
}
