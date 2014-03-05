/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package vbox

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type VMConfig struct {
	Name  string
	Dir   string
	Image string
}

func LaunchVM(c *VMConfig) (*exec.Cmd, error) {
	VBoxManage("createvm", "--name", c.Name, "-ostype", "Linux26_64")
	VBoxManage("registervm", filepath.Join(c.Dir, c.Name, fmt.Sprintf("%s.vbox", c.Name)))
	cmd := exec.Command("cp", c.Image, c.storagePath())
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}
	VBoxManage("storagectl", c.Name, "--name", "SATA", "--add", "sata", "--controller", "IntelAHCI")
	VBoxManage("storageattach", c.Name, "--storagectl", "SATA", "--port", "0", "--type", "hdd", "--medium", c.storagePath())
	err = VBoxManage("modifyvm", c.Name, "--nic1", "hostonly", "--nictype1", "virtio", "--hostonlyadapter1", "vboxnet0")
	if err != nil {
		return nil, err
	}
	err = VBoxManage("modifyvm", c.Name, "--hpet", "on")
	if err != nil {
		return nil, err
	}
	err = VBoxManage("modifyvm", c.Name, "--uart1", "0x3f8", "4", "--uartmode1", "server", c.sockPath())
	if err != nil {
		return nil, err
	}
	cmd, err = VBoxHeadless("--startvm", c.Name)
	if err != nil {
		return nil, err
	}
	time.Sleep(1 * time.Second)
	conn, err := net.Dial("unix", c.sockPath())
	if err != nil {
		return nil, err
	}
	go io.Copy(conn, os.Stdin)
	go io.Copy(os.Stdout, conn)
	return cmd, nil
}

func VBoxManage(args ...string) error {
	cmd := exec.Command("VBoxManage", args...)
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("VBoxManage %s", args)
	}
	return nil
}

func VBoxHeadless(args ...string) (*exec.Cmd, error) {
	cmd := exec.Command("VBoxHeadless", args...)
	err := cmd.Start()
	return cmd, err
}

func (c *VMConfig) sockPath() string {
	return filepath.Join(c.Dir, c.Name, fmt.Sprintf("%s.sock", c.Name))
}

func (c *VMConfig) storagePath() string {
	return filepath.Join(c.Dir, c.Name, fmt.Sprintf("%s.vdi", c.Name))
}
