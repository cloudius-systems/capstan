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
	"github.com/cloudius-systems/capstan/util"
	"io/ioutil"
	"path/filepath"
)

func Instances() error {
	header := fmt.Sprintf("%-35s %-10s %-10s %-15s", "Name", "Platform", "Status", "Image")
	fmt.Println(header)
	rootDir := filepath.Join(util.HomePath(), ".capstan", "instances")
	platforms, _ := ioutil.ReadDir(rootDir)
	for _, platform := range platforms {
		if platform.IsDir() {
			platformDir := filepath.Join(rootDir, platform.Name())
			instances, _ := ioutil.ReadDir(platformDir)
			for _, instance := range instances {
				if instance.IsDir() {
					instanceDir := filepath.Join(platformDir, instance.Name())
					printInstance(instance.Name(), platform.Name(), instanceDir)
				}
			}
		}
	}

	return nil
}

func printInstance(name, platform, dir string) error {
	var status string

	switch platform {
	case "qemu":
		status, _ = qemu.GetVMStatus(name, dir)
	case "vbox":
		status, _ = vbox.GetVMStatus(name, dir)
	case "vmw":
		status, _ = vmw.GetVMStatus(name, dir)
	case "gce":
		status, _ = gce.GetVMStatus(name, dir)
	}
	fmt.Printf("%-35s %-10s %-10s %-15s\n", name, platform, status, "")
	return nil
}
