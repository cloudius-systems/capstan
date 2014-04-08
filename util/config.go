/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"github.com/kylelemons/go-gypsy/yaml"
	"strings"
)

type Config struct {
	Base    string
	RpmBase *RpmPackage
	Cmdline string
	Build   string
	Files   map[string]string
}

func ConfigExists(filename string) bool {
	_, err := yaml.ReadFile(filename)
	return err == nil
}

func ReadConfig(filename string) (*Config, error) {
	config, err := yaml.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseConfig(config)
}

func ParseConfig(config *yaml.File) (*Config, error) {
	base, err := config.Get("base")
	if err != nil {
		return nil, err
	}
	var rpm *RpmPackage = nil
	rpmBaseNode, err := yaml.Child(config.Root, "rpm-base")
	if err != nil {
		return nil, err
	}
	if rpmBaseNode != nil {
		rpmBaseMap := rpmBaseNode.(yaml.Map)
		scalar := rpmBaseMap["name"].(yaml.Scalar)
		name := strings.TrimSpace(scalar.String())
		scalar = rpmBaseMap["version"].(yaml.Scalar)
		version := strings.TrimSpace(scalar.String())
		scalar = rpmBaseMap["release"].(yaml.Scalar)
		release := strings.TrimSpace(scalar.String())
		scalar = rpmBaseMap["arch"].(yaml.Scalar)
		arch := strings.TrimSpace(scalar.String())
		rpm = &RpmPackage{
			Name:    name,
			Version: version,
			Release: release,
			Arch:    arch,
		}
	}
	cmdline, err := config.Get("cmdline")
	if err != nil {
		return nil, err
	}
	build, _ := config.Get("build")
	filesNode, err := yaml.Child(config.Root, "files")
	if err != nil {
		return nil, err
	}
	files := make(map[string]string)
	if filesNode != nil {
		filesMap := filesNode.(yaml.Map)
		for key, value := range filesMap {
			scalar := value.(yaml.Scalar)
			files[key] = strings.TrimSpace(scalar.String())
		}
	}
	result := &Config{
		Base:    base,
		RpmBase: rpm,
		Cmdline: cmdline,
		Build:   build,
		Files:   files,
	}
	return result, nil
}
