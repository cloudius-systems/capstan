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
	"strings"
)

type RunConfig struct {
	InstanceName string
	ImageName    string
	Hypervisor   string
	Verbose      bool
	Memory       string
	Cpus         int
	Networking   string
	Bridge       string
	NatRules     []nat.Rule
}

func Run(repo *util.Repo, config *RunConfig) error {
	var path string
	var cmd *exec.Cmd

	// Start an existing instance
	if config.ImageName == "" && config.InstanceName != "" {
		instanceName, instancePlatform := util.SearchInstance(config.InstanceName)
		if instanceName != "" {
			defer fmt.Println("")

			fmt.Printf("Created instance: %s\n", instanceName)
			// Do not set RawTerm for gce
			if (instancePlatform != "gce") {
				tio, _ := util.RawTerm()
				defer util.ResetTerm(tio)
			}

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
			case "gce":
				c, _ := gce.LoadConfig(instanceName)
				cmd, err = gce.LaunchVM(c)
			}

			if err != nil {
				return err
			}
			if cmd != nil {
				return cmd.Wait()
			}
			return nil
		} else {
			// The InstanceName is actually a ImageName
			// so, cmd like "capstan run cloudius/osv" will work
			config.ImageName = config.InstanceName
			config.InstanceName = strings.Replace(config.InstanceName, "/", "-", -1)
			return Run(repo, config)
		}
	// Both ImageName and InstanceName are specified
	} else if config.ImageName != "" && config.InstanceName != "" {
		if _, err := os.Stat(config.ImageName); os.IsNotExist(err) {
			if repo.ImageExists(config.Hypervisor, config.ImageName) {
				path = repo.ImagePath(config.Hypervisor, config.ImageName)
			} else if image.IsCloudImage(config.ImageName) {
				path = config.ImageName
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
		deleteInstance(config.InstanceName)

	// Valid only when Capstanfile is present
	} else if config.ImageName == "" && config.InstanceName == "" {
		config.ImageName = repo.DefaultImage()
		config.InstanceName = config.ImageName
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
		deleteInstance(config.InstanceName)

		// Cmdline option is not valid
	} else {
		usage()
		return nil
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

	id := config.InstanceName
	fmt.Printf("Created instance: %s\n", id)
	// Do not set RawTerm for gce
	if (config.Hypervisor != "gce") {
		tio, _ := util.RawTerm()
		defer util.ResetTerm(tio)
	}

	switch config.Hypervisor {
	case "qemu":
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
		cmd, err = qemu.LaunchVM(config)
	case "vbox":
		if format != image.VDI && format != image.VMDK {
			return fmt.Errorf("%s: image format of %s is not supported, unable to run it.", config.Hypervisor, path)
		}
		dir := filepath.Join(util.HomePath(), ".capstan/instances/vbox", id)
		config := &vbox.VMConfig{
			Name:       id,
			Dir:        filepath.Join(util.HomePath(), ".capstan/instances/vbox"),
			Image:      path,
			Memory:     size,
			Cpus:       config.Cpus,
			Networking: config.Networking,
			Bridge:     config.Bridge,
			NatRules:   config.NatRules,
			ConfigFile: filepath.Join(dir, "osv.config"),
		}
		cmd, err = vbox.LaunchVM(config)
	case "gce":
		if format != image.GCE_TARBALL && format != image.GCE_GS {
			return fmt.Errorf("%s: image format of %s is not supported, unable to run it.", config.Hypervisor, path)
		}
		dir := filepath.Join(util.HomePath(), ".capstan/instances/gce", id)
		bucket := "osvimg"
		config := &gce.VMConfig{
			Name:             id,
			Image:		  id,
			Network:          "default",
			MachineType:      "n1-standard-1",
			Zone:             "us-central1-a",
			ConfigFile:	  filepath.Join(dir, "osv.config"),
			InstanceDir:	  dir,
		}
		if format == image.GCE_TARBALL {
			config.CloudStoragePath = "gs://" + bucket + "/" + id + ".tar.gz"
			config.Tarball = path
		} else {
			config.CloudStoragePath = path
			config.Tarball = ""
		}
		cmd, err = gce.LaunchVM(config)
	case "vmw":
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

func usage() {
	fmt.Println("Please try one of the following:")
	fmt.Println("1) capstan run")
	fmt.Println("   run under a directory contains Capstanfile")
	fmt.Println("2) capstan run $instance_name")
	fmt.Println("   start an existing instance")
	fmt.Println("3) capstan run -i $image_name $instance_name")
	fmt.Println("   start an instance using $image_name")
}

func deleteInstance(name string) error {
	instanceName, instancePlatform := util.SearchInstance(name)
	if instanceName == "" {
		return nil
	}
	var err error
	switch instancePlatform {
	case "qemu":
		err = qemu.DeleteVM(name)
	case "vbox":
		err = vbox.DeleteVM(name)
	case "vmw":
		err = vmw.DeleteVM(name)
	case "gce":
		err = gce.DeleteVM(name)
	}
	return err
}
