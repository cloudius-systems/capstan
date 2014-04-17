/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package gce

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type VMConfig struct {
	Name             string
	Image            string
	Network          string
	MachineType      string
	Zone             string
	CloudStoragePath string
	Tarball          string
}

func LaunchVM(c *VMConfig) (*exec.Cmd, error) {
	err := vmUploadImage(c)
	if err != nil {
		return nil, err
	}
	defer vmDeleteImage(c)
	err = vmCreate(c)
	if err != nil {
		return nil, err
	}
	err = vmPrintInfo(c)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func vmCreate(c *VMConfig) error {
	err := gcUtil("addinstance", "--image", c.Image, "--network", c.Network, "--machine_type", c.MachineType, "--zone", c.Zone, c.Name)
	if err != nil {
		return err
	}
	return nil
}

func vmDeleteImage(c *VMConfig) error {
	err := gsUtil("rm", c.CloudStoragePath)
	if err != nil {
		return err
	}
	err = gcUtil("deleteimage", "-f", c.Image)
	if err != nil {
		return err
	}
	return nil
}

func vmUploadImage(c *VMConfig) error {
	err := gsUtil("cp", c.Tarball, c.CloudStoragePath)
	if err != nil {
		return err
	}
	err = gcUtil("addimage", c.Image, c.CloudStoragePath)
	if err != nil {
		return err
	}
	return nil
}

func vmGetIP(c *VMConfig) (string, string, error) {
	externalIP, internalIP := "Unknow", "Unknow"
	cmd := exec.Command("gcutil", "getinstance", c.Name)
	out, err := cmd.Output()
	if err != nil {
		return externalIP, internalIP, nil
	}

	lines := strings.Split(string(out), "\n")
	r, _ := regexp.Compile("\\|(.*)\\|(.*)\\|")

	for _, line := range lines {
		match := r.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		key := strings.TrimSpace(match[1])
		value := strings.TrimSpace(match[2])

		if key == "external-ip" {
			externalIP = value
		} else if key == "ip" {
			internalIP = value
		}
	}

	return externalIP, internalIP, nil
}

func vmPrintInfo(c *VMConfig) error {
	externalIP, internalIP, err := vmGetIP(c)
	if err != nil {
		fmt.Printf("Failed To Get Instance IP Info: %s\n", c.Name)
		return err
	}

	fmt.Printf("Created Instance: %s\n", c.Name)
	fmt.Printf("Public        IP: %s\n", externalIP)
	fmt.Printf("Internal      IP: %s\n", internalIP)
	fmt.Printf("Machine     Type: %s\n", c.MachineType)
	fmt.Printf("Zone            : %s\n", c.Zone)

	return nil
}

func DeleteVM(name string) error {
	return gcUtil("deleteinstance", "--delete_boot_pd", "-f", name)
}

func gsUtil(args ...string) error {
	cmd := exec.Command("gsutil", args...)
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("gcutil %s", args)
	}
	return nil
}

func gcUtil(args ...string) error {
	cmd := exec.Command("gcutil", args...)
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("gcutil %s", args)
	}
	return nil
}
