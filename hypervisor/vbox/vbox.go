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
	"regexp"
	"strconv"
	"strings"
	"time"
)

type VMConfig struct {
	Name   string
	Dir    string
	Image  string
	Memory int64
	Cpus   int
}

func LaunchVM(c *VMConfig) (*exec.Cmd, error) {
	exists, err := vmExists(c.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		err := vmDelete(c)
		if err != nil {
			return nil, err
		}
	}
	err = vmCreate(c)
	if err != nil {
		return nil, err
	}
	cmd, err := VBoxHeadless("--startvm", c.Name)
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

func vmExists(vmName string) (bool, error) {
	vms, err := vmList()
	if err != nil {
		return false, err
	}
	for _, vm := range vms {
		if vm == vmName {
			return true, nil
		}
	}
	return false, nil
}

func vmList() ([]string, error) {
	cmd := exec.Command("VBoxManage", "list", "vms")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	vms := make([]string, 0)
	lines := strings.Split(string(out), "\n")
	r, _ := regexp.Compile("\"(.*)\"")
	for _, line := range lines {
		vm := r.FindStringSubmatch(line)
		if len(vm) > 0 {
			vms = append(vms, vm[1])
		}
	}
	return vms, nil
}

func vmCreate(c *VMConfig) error {
	err := VBoxManage("createvm", "--name", c.Name, "-ostype", "Linux26_64")
	if err != nil {
		return err
	}
	err = VBoxManage("registervm", filepath.Join(c.Dir, c.Name, fmt.Sprintf("%s.vbox", c.Name)))
	if err != nil {
		return err
	}
	cmd := exec.Command("cp", c.Image, c.storagePath())
	if err := cmd.Run(); err != nil {
		return err
	}
	err = VBoxManage("storagectl", c.Name, "--name", "SATA", "--add", "sata", "--controller", "IntelAHCI")
	if err != nil {
		return err
	}
	err = VBoxManage("storageattach", c.Name, "--storagectl", "SATA", "--port", "0", "--type", "hdd", "--medium", c.storagePath())
	if err != nil {
		return err
	}
	err = VBoxManage("modifyvm", c.Name, "--nic1", "hostonly", "--nictype1", "virtio", "--hostonlyadapter1", "vboxnet0")
	if err != nil {
		return err
	}
	err = VBoxManage("modifyvm", c.Name, "--hpet", "on")
	if err != nil {
		return err
	}
	err = VBoxManage("modifyvm", c.Name, "--uart1", "0x3f8", "4", "--uartmode1", "server", c.sockPath())
	if err != nil {
		return err
	}
	err = VBoxManage("modifyvm", c.Name, "--memory", strconv.FormatInt(c.Memory, 10))
	if err != nil {
		return err
	}
	err = VBoxManage("modifyvm", c.Name, "--cpus", strconv.Itoa(c.Cpus))
	if err != nil {
		return err
	}
	return nil
}

func vmDelete(c *VMConfig) error {
	return VBoxManage("unregistervm", c.Name, "--delete")
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
