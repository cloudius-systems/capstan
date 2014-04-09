/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan/util"
	"github.com/cloudius-systems/capstan/hypervisor/qemu"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"errors"
)

func Build(r *util.Repo, hypervisor string, image string, verbose bool) error {
	config, err := util.ReadConfig("Capstanfile")
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
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return err
		}
	}
	err = checkConfig(config, r, hypervisor)
	if err != nil {
		return err
	}
	cmd := util.CopyFile(r.ImagePath(hypervisor, config.Base), r.ImagePath(hypervisor, image))
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

func checkConfig(config *util.Config, r *util.Repo, hypervisor string) error {
	if _, err := os.Stat(r.ImagePath(hypervisor, config.Base)); os.IsNotExist(err) {
		err := Pull(r, hypervisor, config.Base)
		if err != nil {
			return err
		}
	}
	for _, value := range config.Files {
		if _, err := os.Stat(value); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("%s: no such file or directory", value))
		}
	}
	return nil
}
