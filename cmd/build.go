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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Build(r *capstan.Repo, hypervisor string, image string, verbose bool) error {
	config, err := capstan.ReadConfig("Capstanfile")
	if err != nil {
		return err
	}
	fmt.Printf("Building %s...\n", image)
	err = os.MkdirAll(filepath.Dir(r.ImagePath(hypervisor, image)), 0777)
	if err != nil {
		return err
	}
	if config.RpmBase != nil {
		config.RpmBase.Download()
	}
	if config.Build != "" {
		args := strings.Fields(config.Build)
		cmd := exec.Command(args[0], args[1:]...)
		_, err = cmd.Output()
		if err != nil {
			return err
		}
	}
	err = config.Check(r, hypervisor)
	if err != nil {
		return err
	}
	cmd := exec.Command("cp", r.ImagePath(hypervisor, config.Base), r.ImagePath(hypervisor, image))
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	err = qemu.SetArgs(r, hypervisor, image, "/tools/cpiod.so")
	if err != nil {
		return err
	}
	if config.RpmBase != nil {
		err = qemu.UploadRPM(r, hypervisor, image, config, verbose)
		if err != nil {
			return err
		}
	}
	err = qemu.UploadFiles(r, hypervisor, image, config, verbose)
	if err != nil {
		return err
	}
	err = qemu.SetArgs(r, hypervisor, image, config.Cmdline)
	if err != nil {
		return err
	}
	return nil
}
