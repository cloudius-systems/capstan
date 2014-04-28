/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package vbox

import (
	"fmt"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/util"
	"gopkg.in/yaml.v1"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type VMConfig struct {
	Name       string
	Dir        string
	Image      string
	Memory     int64
	Cpus       int
	Networking string
	Bridge      string
	NatRules   []nat.Rule
	ConfigFile string
}

func LaunchVM(c *VMConfig) (*exec.Cmd, error) {
	exists, err := vmExists(c.Name)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = vmCreate(c)
		if err != nil {
			return nil, err
		}
	}

	StoreConfig(c)

	cmd, err := VBoxHeadless("--startvm", c.Name)
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	for i := 0; i < 5; i++ {
		conn, err = util.Connect(c.sockPath())
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		return nil, err
	}
	go io.Copy(conn, os.Stdin)
	go io.Copy(os.Stdout, conn)
	return cmd, nil
}

func vmExists(vmName string) (bool, error) {
	vms, err := vmList("vms")
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

func vmList(list_type string) ([]string, error) {
	cmd := exec.Command("VBoxManage", "list", list_type)
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
	err := VBoxManage("createvm", "--name", c.Name, "--basefolder", c.Dir, "-ostype", "Linux26_64")
	if err != nil {
		return err
	}
	err = VBoxManage("registervm", filepath.Join(c.Dir, c.Name, fmt.Sprintf("%s.vbox", c.Name)))
	if err != nil {
		return err
	}
	err = VBoxManage("clonehd", c.Image, c.storagePath())
	if err != nil {
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
	err = vmSetupNetworking(c)
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
	err = VBoxManage("setextradata", c.Name, "VBoxInternal/CPUM/SSE4.1", "1")
	if err != nil {
		return err
	}
	return nil
}

func vmSetupNetworking(c *VMConfig) error {
	switch c.Networking {
	case "bridge":
		return VBoxManage("modifyvm", c.Name, "--nic1", "bridged", "--bridgeadapter1", c.Bridge, "--nictype1", "virtio")
	case "nat":
		return vmSetupNAT(c)
	}
	return fmt.Errorf("%s: networking not supported", c.Networking)
}

func vmSetupNAT(c *VMConfig) error {
	err := VBoxManage("modifyvm", c.Name, "--nic1", "nat", "--nictype1", "virtio")
	if err != nil {
		return err
	}
	for _, rule := range c.NatRules {
		natRule := fmt.Sprintf("guest%s,tcp,,%s,,%s", rule.GuestPort, rule.HostPort, rule.GuestPort)
		err := VBoxManage("modifyvm", c.Name, "--natpf1", natRule)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteVM(name string) error {
	dir := filepath.Join(util.HomePath(), ".capstan/instances/vbox", name)
	c := &VMConfig{
		ConfigFile: filepath.Join(dir, "osv.config"),
	}

	cmd := exec.Command("rm", "-f", c.ConfigFile)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to delete: %s", c.ConfigFile)
		return err
	}

	return VBoxManage("unregistervm", name, "--delete")
}

func StopVM(name string) error {
	return VBoxManage("controlvm", name, "poweroff")
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
	if runtime.GOOS == "windows" {
		return "\\\\.\\pipe\\" + c.Name
	} else {
		return filepath.Join(c.Dir, c.Name, fmt.Sprintf("%s.sock", c.Name))
	}
}

func (c *VMConfig) storagePath() string {
	return filepath.Join(c.Dir, c.Name, "disk.vdi")
}

func LoadConfig(name string) (*VMConfig, error) {
	dir := filepath.Join(util.HomePath(), ".capstan/instances/vbox", name)
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

func StoreConfig(c *VMConfig) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.ConfigFile, data, 0644)
}

func GetVMStatus(name, dir string) (string, error) {
	vms, err := vmList("runningvms")
	if err != nil {
		return "Stopped", err
	}
	for _, vm := range vms {
		if vm == name {
			return "Running", nil
		}
	}
	return "Stopped", nil
}
