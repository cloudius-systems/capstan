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
)

type Config struct {
	Base    string
	RpmBase *RpmPackage "rpm-base"
	Cmdline string
	Build   string
	Files   map[string]string
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
	return ParseConfig(data)
}

func ParseConfig(data []byte) (*Config, error) {
	c := Config{}
	err := yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	if c.Cmdline == "" {
		return nil, fmt.Errorf("\"cmdline\" not found")
	}
	return &c, nil
}
