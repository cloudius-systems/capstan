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
)

func Stop(name string) error {
	instanceName, instancePlatform := util.SearchInstance(name)
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
	case "gce":
		err = gce.StopVM(name)
	}

	if err != nil {
		fmt.Printf("Failed to stop instance: %s\n", name)
	}

	fmt.Printf("Stopped instance: %s\n", name)
	return nil
}
