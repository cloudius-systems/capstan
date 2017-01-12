/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"fmt"
	"github.com/mikelangelo-project/capstan/hypervisor/gce"
	"github.com/mikelangelo-project/capstan/hypervisor/qemu"
	"github.com/mikelangelo-project/capstan/hypervisor/vbox"
	"github.com/mikelangelo-project/capstan/hypervisor/vmw"
	"github.com/mikelangelo-project/capstan/util"
	"io/ioutil"
	"os"
	"path/filepath"
)

func Instances() error {
	header := fmt.Sprintf("%-35s %-10s %-10s %-15s", "Name", "Platform", "Status", "Image")
	fmt.Println(header)
	rootDir := filepath.Join(util.ConfigDir(), "instances")
	platforms, _ := ioutil.ReadDir(rootDir)
	for _, platform := range platforms {
		if platform.IsDir() {
			platformDir := filepath.Join(rootDir, platform.Name())
			instances, _ := ioutil.ReadDir(platformDir)
			for _, instance := range instances {
				if instance.IsDir() {
					instanceDir := filepath.Join(platformDir, instance.Name())

					// Instance only exists if osv.config is present.
					if _, err := os.Stat(filepath.Join(instanceDir, "osv.config")); os.IsNotExist(err) {
						continue
					}

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
