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
)

func Delete(name string) error {
	instanceName, instancePlatform := util.SearchInstance(name)
	if instanceName == "" {
		fmt.Printf("Instance: %s not found\n", name)
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

	if err != nil {
		fmt.Printf("Failed to delete instance: %s\n", name)
		return err
	}

	fmt.Printf("Deleted instance: %s\n", name)
	return nil
}
