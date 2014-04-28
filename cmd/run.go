/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan/hypervisor/gce"
	"github.com/cloudius-systems/capstan/hypervisor/qemu"
	"github.com/cloudius-systems/capstan/hypervisor/vbox"
	"github.com/cloudius-systems/capstan/hypervisor/vmw"
	"github.com/cloudius-systems/capstan/image"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/util"
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
	Networking string
	Bridge     string
	NatRules   []nat.Rule
}

func Run(repo *util.Repo, config *RunConfig) error {
	var path string
	var cmd *exec.Cmd

	instanceName, instancePlatform := util.SearchInstance(config.ImageName)
	if instanceName != "" {
		fmt.Printf("Created instance: %s\n", instanceName)
		defer fmt.Println("")

		tio, _ := util.RawTerm()
		defer util.ResetTerm(tio)

		var err error
		switch instancePlatform {
		case "qemu":
			c, _ := qemu.LoadConfig(instanceName)
			cmd, err = qemu.LaunchVM(c)
		case "vbox":
			c, _ := vbox.LoadConfig(instanceName)
			cmd, err = vbox.LaunchVM(c)
		case "vmw":
			c, _ := vmw.LoadConfig(instanceName)
			cmd, err = vmw.LaunchVM(c)
		}

		if err != nil {
			return err
		}
		if cmd != nil {
			return cmd.Wait()
		}
		return nil
	}
	if config.ImageName != "" {
		if _, err := os.Stat(config.ImageName); os.IsNotExist(err) {
			if repo.ImageExists(config.Hypervisor, config.ImageName) {
				path = repo.ImagePath(config.Hypervisor, config.ImageName)
			} else if util.IsRemoteImage(config.ImageName) {
				err := Pull(repo, config.Hypervisor, config.ImageName)
				if err != nil {
					return err
				}
				path = repo.ImagePath(config.Hypervisor, config.ImageName)
			} else {
				return fmt.Errorf("%s: no such image", config.ImageName)
			}
		} else {
			path = config.ImageName
		}
	} else {
		config.ImageName = repo.DefaultImage()
		if config.ImageName == "" {
			return fmt.Errorf("No Capstanfile found, unable to run.")
		}
		if !repo.ImageExists(config.Hypervisor, config.ImageName) {
			if !util.ConfigExists("Capstanfile") {
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
	size, err := util.ParseMemSize(config.Memory)
	if err != nil {
		return err
	}
	defer fmt.Println("")
	switch config.Hypervisor {
	case "qemu":
		id := util.ID()
		dir := filepath.Join(os.Getenv("HOME"), ".capstan/instances/qemu", id)
		config := &qemu.VMConfig{
			Name:        id,
			Image:       path,
			Verbose:     true,
			Memory:      size,
			Cpus:        config.Cpus,
			Networking:  config.Networking,
			Bridge:      config.Bridge,
			NatRules:    config.NatRules,
			BackingFile: true,
			InstanceDir: dir,
			Monitor:     filepath.Join(dir, "osv.monitor"),
			ConfigFile:  filepath.Join(dir, "osv.config"),
		}
		fmt.Printf("Created instance: %s\n", id)
		tio, _ := util.RawTerm()
		defer util.ResetTerm(tio)
		cmd, err = qemu.LaunchVM(config)
	case "vbox":
		if format != image.VDI && format != image.VMDK {
			return fmt.Errorf("%s: image format of %s is not supported, unable to run it.", config.Hypervisor, path)
		}
		id := util.ID()
		dir := filepath.Join(util.HomePath(), ".capstan/instances/vbox", id)
		config := &vbox.VMConfig{
			Name:       id,
			Dir:        filepath.Join(util.HomePath(), ".capstan/instances/vbox"),
			Image:      path,
			Memory:     size,
			Cpus:       config.Cpus,
			Networking: config.Networking,
			NatRules:   config.NatRules,
			ConfigFile: filepath.Join(dir, "osv.config"),
		}
		fmt.Printf("Created instance: %s\n", id)
		tio, _ := util.RawTerm()
		defer util.ResetTerm(tio)
		cmd, err = vbox.LaunchVM(config)
	case "gce":
		id := util.ID()
		bucket := "osvimg"
		config := &gce.VMConfig{
			Name:             "osv-capstan-" + id,
			Image:            "osv-capstan-" + id,
			Network:          "default",
			MachineType:      "n1-standard-1",
			Zone:             "us-central1-a",
			CloudStoragePath: "gs://" + bucket + "/osv-capstan-" + id + ".tar.gz",
			Tarball:          path,
		}
		cmd, err = gce.LaunchVM(config)
	case "vmw":
		id := util.ID()
		if format != image.VMDK {
			return fmt.Errorf("%s: image format of %s is not supported, unable to run it.", config.Hypervisor, path)
		}
		dir := filepath.Join(util.HomePath(), ".capstan/instances/vmw", id)
		config := &vmw.VMConfig{
			Name:         id,
			Dir:          dir,
			Image:        filepath.Join(dir, "osv.vmdk"),
			Memory:       size,
			Cpus:         config.Cpus,
			NatRules:     config.NatRules,
			VMXFile:      filepath.Join(dir, "osv.vmx"),
			InstanceDir:  dir,
			OriginalVMDK: path,
			ConfigFile:   filepath.Join(dir, "osv.config"),
		}
		fmt.Printf("Created instance: %s\n", id)
		tio, _ := util.RawTerm()
		defer util.ResetTerm(tio)
		cmd, err = vmw.LaunchVM(config)
	default:
		err = fmt.Errorf("%s: is not a supported hypervisor", config.Hypervisor)
	}
	if err != nil {
		return err
	}
	if cmd != nil {
		return cmd.Wait()
	} else {
		return nil
	}
}
