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
	"github.com/cloudius-systems/capstan/hypervisor/vbox"
	"github.com/cloudius-systems/capstan/image"
	"os"
	"os/exec"
	"path/filepath"
)

type RunConfig struct {
	ImageName  string
	Hypervisor string
	Verbose    bool
}

func Run(repo *capstan.Repo, config *RunConfig) error {
	var path string
	if _, err := os.Stat(config.ImageName); os.IsNotExist(err) {
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
	} else {
		path = config.ImageName
	}
	format, err := image.Probe(path)
	if err != nil {
		return err
	}
	if format == image.Unknown {
		return fmt.Errorf("%s: image format not recognized, unable to run it.", path)
	}
	var cmd *exec.Cmd
	switch config.Hypervisor {
	case "kvm":
		cmd, err = qemu.LaunchVM(true, path)
	case "vbox":
		config := &vbox.VMConfig{
			Name:  "osv",
			Dir:   filepath.Join(os.Getenv("HOME"), "VirtualBox VMs"),
			Image: path,
		}
		cmd, err = vbox.LaunchVM(config)
	default:
		err = fmt.Errorf("%s: is not a supported hypervisor", config.Hypervisor)
	}
	if err != nil {
		return err
	}
	return cmd.Wait()
}
