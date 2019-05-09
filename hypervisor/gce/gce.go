/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package gce

import (
	"fmt"
	"github.com/cloudius-systems/capstan/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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
	ConfigFile       string
	InstanceDir      string
	BootDisk         string
}

func LaunchVM(c *VMConfig) (*exec.Cmd, error) {
	err := LoginCheck()
	if err != nil {
		return nil, err
	}

	err = vmUploadImage(c)
	if err != nil {
		return nil, err
	}
	defer vmDeleteImage(c)
	err = vmCreate(c)
	if err != nil {
		return nil, err
	}

	StoreConfig(c)

	err = vmPrintInfo(c)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func StopVM(name string) error {
	return gcUtil("deleteinstance", "--nodelete_boot_pd", "-f", name)
}

func vmCreate(c *VMConfig) error {
	var err error
	if c.BootDisk == "" {
		err = gcUtil("addinstance", "--image", c.Image, "--network", c.Network, "--machine_type", c.MachineType, "--zone", c.Zone, c.Name)
		c.BootDisk = c.Image
	} else {
		err = gcUtil("addinstance", "--disk", c.BootDisk+",boot", "--network", c.Network, "--machine_type", c.MachineType, "--zone", c.Zone, c.Name)
	}
	if err != nil {
		return err
	}
	return nil
}

func vmDeleteImage(c *VMConfig) error {
	if c.Tarball != "" {
		c.Tarball = ""
		err := gsUtil("rm", c.CloudStoragePath)
		if err != nil {
			return err
		}
	}
	err := gcUtil("deleteimage", "-f", c.Image)
	if err != nil {
		return err
	}
	return nil
}

func vmUploadImage(c *VMConfig) error {
	if c.Tarball != "" {
		err := gsUtil("cp", c.Tarball, c.CloudStoragePath)
		if err != nil {
			return err
		}
	}
	err := gcUtil("addimage", c.Image, c.CloudStoragePath)
	if err != nil {
		return err
	}
	return nil
}

func vmGetInfo(name string) (string, string, string, error) {
	status, externalIP, internalIP := "", "Unknow", "Unknow"
	cmd := exec.Command("gcutil", "getinstance", name)
	out, err := cmd.Output()
	if err != nil {
		return status, externalIP, internalIP, nil
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
		} else if key == "status" {
			status = value
		}
	}

	return status, externalIP, internalIP, nil
}

func vmPrintInfo(c *VMConfig) error {
	_, externalIP, internalIP, err := vmGetInfo(c.Name)
	if err != nil {
		fmt.Printf("Failed To Get Instance IP Info: %s\n", c.Name)
		return err
	}

	fmt.Printf("Public        IP: %s\n", externalIP)
	fmt.Printf("Internal      IP: %s\n", internalIP)
	fmt.Printf("Machine     Type: %s\n", c.MachineType)
	fmt.Printf("Zone            : %s\n", c.Zone)
	fmt.Printf("Serial          : gcutil getserialportoutput %s\n", c.Name)
	fmt.Printf("SSH             : gcutil ssh %s\n", c.Name)
	fmt.Printf("SSH             : ssh admin@%s\n", externalIP)

	return nil
}

func DeleteVM(name string) error {
	gcUtil("deleteinstance", "--delete_boot_pd", "-f", name)

	gcUtil("deletedisk", "-f", name)

	dir := filepath.Join(util.ConfigDir(), "instances/gce", name)
	c := &VMConfig{
		InstanceDir: dir,
		ConfigFile:  filepath.Join(dir, "osv.config"),
	}
	cmd := exec.Command("rm", "-f", c.ConfigFile)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("rm failed: %s", c.ConfigFile)
		return err
	}

	cmd = exec.Command("rmdir", c.InstanceDir)
	_, err = cmd.Output()
	if err != nil {
		fmt.Printf("rmdir failed: %s", c.InstanceDir)
		return err
	}

	return nil
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

func GetVMStatus(name, dir string) (string, error) {
	status, _, _, err := vmGetInfo(name)
	if err != nil {
		return "Stopped", err
	}

	if status == "RUNNING" {
		return "Running", nil
	} else if status == "STOPPING" {
		return "Running", nil
	} else {
		return "Stopped", nil
	}
}

func StoreConfig(c *VMConfig) error {
	dir := c.InstanceDir
	err := os.MkdirAll(dir, 0775)
	if err != nil {
		fmt.Printf("mkdir failed: %s", dir)
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.ConfigFile, data, 0644)
}

func LoadConfig(name string) (*VMConfig, error) {
	dir := filepath.Join(util.ConfigDir(), "instances/gce", name)
	file := filepath.Join(dir, "osv.config")
	c := VMConfig{}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Failed to open: %s\n", file)
		return nil, err
	}
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func LoginCheck() error {
	_, err := exec.LookPath("gcutil")
	if err != nil {
		fmt.Println("gcutil is not found. Please install Google Cloud SDK:")
		fmt.Println("https://developers.google.com/cloud/sdk")
		return err
	}
	out, err := exec.Command("gcutil", "whoami").CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return err
	}
	return nil
}
