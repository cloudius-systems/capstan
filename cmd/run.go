/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan"
	"github.com/cloudius-systems/capstan/hypervisor/qemu"
	"github.com/cloudius-systems/capstan/image"
	"os"
)

type RunConfig struct {
	ImageName  string
	Hypervisor string
	Verbose    bool
}

func Run(repo *capstan.Repo, config *RunConfig) error {
	if config.Hypervisor != "kvm" {
		return fmt.Errorf("%s: is not a supported hypervisor", config.Hypervisor)
	}
	var path string
	file, err := os.Open(config.ImageName)
	if err == nil {
		path = config.ImageName
		format := image.Probe(file)
		if format == image.Unknown {
			file.Close()
			return fmt.Errorf("%s: image format not recognized, unable to run it.", path)
		}
		file.Close()
	} else {
		if !repo.ImageExists(config.ImageName) {
			if !capstan.ConfigExists("Capstanfile") {
				return fmt.Errorf("%s: no such image", config.ImageName)
			}
			err := qemu.BuildImage(repo, config.ImageName, config.Verbose)
			if err != nil {
				return err
			}
		}
		path = repo.ImagePath(config.ImageName)
	}
	cmd, err := qemu.LaunchVM(true, path)
	if err != nil {
		return err
	}
	cmd.Wait()
	return nil
}
