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

func Delete(name string) error {
	var err error
	instanceName, instancePlatform := util.SearchInstance(name)
	if instanceName == "" {
		fmt.Printf("Instance: %s not found\n", name)
		return nil
	}

	switch instancePlatform {
	case "qemu":
		qemu.StopVM(name)
		err = qemu.DeleteVM(name)
	case "vbox":
		vbox.StopVM(name)
		err = vbox.DeleteVM(name)
	case "vmw":
		vmw.StopVM(name)
		err = vmw.DeleteVM(name)
	case "gce":
		gce.StopVM(name)
		err = gce.DeleteVM(name)
	}
	if err != nil {
		fmt.Printf("Failed to delete instance: %s\n", name)
		return err
	}

	fmt.Printf("Deleted instance: %s\n", name)
	return nil
}
