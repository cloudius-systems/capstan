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

func Build(r *capstan.Repo, image string, verbose bool) error {
	config, err := capstan.ReadConfig("Capstanfile")
	if err != nil {
		return err
	}
	fmt.Printf("Building %s...\n", image)
	err = os.MkdirAll(filepath.Dir(r.ImagePath(image)), 0777)
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
	err = config.Check(r)
	if err != nil {
		return err
	}
	cmd := exec.Command("cp", r.ImagePath(config.Base), r.ImagePath(image))
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	err = qemu.SetArgs(r, image, "/tools/cpiod.so")
	if err != nil {
		return err
	}
	if config.RpmBase != nil {
		qemu.UploadRPM(r, image, config, verbose)
	}
	qemu.UploadFiles(r, image, config, verbose)
	err = qemu.SetArgs(r, image, config.Cmdline)
	if err != nil {
		return err
	}
	return nil
}
