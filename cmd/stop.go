/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan/hypervisor/qemu"
	"github.com/cloudius-systems/capstan/hypervisor/vbox"
	"github.com/cloudius-systems/capstan/hypervisor/vmw"
	"github.com/cloudius-systems/capstan/util"
	"io/ioutil"
	"path/filepath"
)

func Stop(name string) error {
	instanceName := ""
	instancePlatform := ""
	rootDir := filepath.Join(util.HomePath(), ".capstan", "instances")
	platforms, _ := ioutil.ReadDir(rootDir)
	for _, platform := range platforms {
		if platform.IsDir() {
			platformDir := filepath.Join(rootDir, platform.Name())
			instances, _ := ioutil.ReadDir(platformDir)
			for _, instance := range instances {
				if instance.IsDir() {
					if name == instance.Name() {
						instanceName = instance.Name()
						instancePlatform = platform.Name()
					}
				}
			}
		}
	}

	if instanceName == "" {
		fmt.Printf("Instance: %s not found\n", name)
		return nil
	}

	var err error
	switch instancePlatform {
	case "qemu":
		err = qemu.StopVM(name)
	case "vbox":
		err = vbox.StopVM(name)
	case "vmw":
		err = vmw.StopVM(name)
	}

	if err != nil {
		fmt.Printf("Failed to stop instance: %s\n", name)
	}

	fmt.Printf("Stopped instance: %s\n", name)
	return nil
}
