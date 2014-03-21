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
	"github.com/cloudius-systems/capstan/nat"
	"os"
	"os/exec"
	"path/filepath"
)

type RunConfig struct {
	ImageName  string
	Hypervisor string
	Verbose    bool
	Memory     string
	Cpus       int
	NatRules   []nat.Rule
}

func Run(repo *capstan.Repo, config *RunConfig) error {
	var path string
	if config.ImageName != "" {
		if _, err := os.Stat(config.ImageName); os.IsNotExist(err) {
			if !repo.ImageExists(config.Hypervisor, config.ImageName) {
				return fmt.Errorf("%s: no such image", config.ImageName)
			}
			path = repo.ImagePath(config.Hypervisor, config.ImageName)
		} else {
			path = config.ImageName
		}
	} else {
		config.ImageName = repo.DefaultImage()
		if config.ImageName == "" {
			return fmt.Errorf("No Capstanfile found, unable to run.")
		}
		if !repo.ImageExists(config.Hypervisor, config.ImageName) {
			if !capstan.ConfigExists("Capstanfile") {
				return fmt.Errorf("%s: no such image", config.ImageName)
			}
			err := Build(repo, config.Hypervisor, config.ImageName, config.Verbose)
			if err != nil {
				return err
			}
		}
		path = repo.ImagePath(config.Hypervisor, config.ImageName)
	}
	format, err := image.Probe(path)
	if err != nil {
		return err
	}
	if format == image.Unknown {
		return fmt.Errorf("%s: image format not recognized, unable to run it.", path)
	}
	size, err := capstan.ParseMemSize(config.Memory)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch config.Hypervisor {
	case "qemu":
		config := &qemu.VMConfig{
			Image:    path,
			Verbose:  true,
			Memory:   size,
			Cpus:     config.Cpus,
			NatRules: config.NatRules,
		}
		cmd, err = qemu.LaunchVM(config)
	case "vbox":
		if format != image.VDI && format != image.VMDK {
			return fmt.Errorf("%s: image format of %s is not supported, unable to run it.", config.Hypervisor, path)
		}
		config := &vbox.VMConfig{
			Name:     "osv",
			Dir:      filepath.Join(os.Getenv("HOME"), "VirtualBox VMs"),
			Image:    path,
			Memory:   size,
			Cpus:     config.Cpus,
			NatRules: config.NatRules,
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
