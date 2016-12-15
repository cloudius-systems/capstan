/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package vmw

import (
	"bufio"
	"fmt"
	"github.com/mikelangelo-project/capstan/nat"
	"github.com/mikelangelo-project/capstan/util"
	"gopkg.in/yaml.v1"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type VMConfig struct {
	Name         string
	Dir          string
	Image        string
	Memory       int64
	Cpus         int
	NatRules     []nat.Rule
	VMXFile      string
	InstanceDir  string
	OriginalVMDK string
	ConfigFile   string
}

func vmxRun(args ...string) (*exec.Cmd, error) {
	if runtime.GOOS == "darwin" {
		path := os.Getenv("PATH")
		path += `:/Applications/VMware Fusion.app/Contents/Library`
		os.Setenv("PATH", path)
	}
	cmd := exec.Command("vmrun", args...)
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func doRead(done chan bool, conn net.Conn) {
	io.Copy(os.Stdout, conn)
	done <- true
}

func LaunchVM(c *VMConfig) (*exec.Cmd, error) {
	if _, err := os.Stat(c.VMXFile); os.IsNotExist(err) {
		dir := c.InstanceDir
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			fmt.Printf("mkdir failed: %s", dir)
			return nil, err
		}
		cmd := util.CopyFile(c.OriginalVMDK, c.Image)
		_, err = cmd.Output()
		if err != nil {
			fmt.Printf("cp failed: %s", c.OriginalVMDK)
			return nil, err
		}
		err = vmCreateVMXFile(c)
		if err != nil {
			fmt.Printf("Create VMXFile failed: %s", c.VMXFile)
			return nil, err
		}
	}

	StoreConfig(c)

	cmd, err := vmxRun("-T", "ws", "start", c.VMXFile, "nogui")
	if err != nil {
		return nil, err
	}

	conn, err := util.ConnectAndWait("unix", c.sockPath())
	if err != nil {
		return nil, err
	}

	done := make(chan bool)
	go io.Copy(conn, os.Stdin)
	go doRead(done, conn)

	// Wait until the serial connection is disconnected
	<-done

	return cmd, nil
}

func DeleteVM(name string) error {
	dir := filepath.Join(util.ConfigDir(), "instances/vmw", name)
	c := &VMConfig{
		VMXFile:    filepath.Join(dir, "osv.vmx"),
		ConfigFile: filepath.Join(dir, "osv.config"),
	}

	cmd := exec.Command("rm", "-f", c.ConfigFile)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to delete: %s", c.ConfigFile)
		return err
	}

	cmd, err = vmxRun("-T", "ws", "deleteVM", c.VMXFile)
	if err != nil {
		fmt.Printf("Failed to delete VM %s", c.VMXFile)
		return err
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Failed to delete VM %s", c.VMXFile)
		return err
	}
	return nil
}

func StopVM(name string) error {
	dir := filepath.Join(util.ConfigDir(), "instances/vmw", name)
	c := &VMConfig{
		VMXFile: filepath.Join(dir, "osv.vmx"),
	}
	cmd, err := vmxRun("-T", "ws", "stop", c.VMXFile)
	if err != nil {
		fmt.Printf("Failed to stop VM %s", c.VMXFile)
		return err
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Failed to stop VM %s", c.VMXFile)
		return err
	}
	return nil
}

func (c *VMConfig) sockPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\` + c.Name
	} else {
		return filepath.Join(c.Dir, "osv.sock")
	}
}

var vmx string = `#!/usr/bin/vmware
config.version = "8"
virtualHW.version = "8"
virtualHW.productCompatibility = "hosted"
guestOS = "ubuntu-64"
nvram = "osv.nvram"
extendedConfigFile = "osv.vmxf"

vcpu.hotadd = "TRUE"
mem.hotadd = "TRUE"

scsi0.present = "TRUE"
scsi0.virtualDev = "pvscsi"
scsi0:0.present = "TRUE"
scsi0:0.fileName = "osv.vmdk"
scsi0:0.redo = ""

ethernet0.present = "TRUE"
ethernet0.connectionType = "nat"
ethernet0.virtualDev = "vmxnet3"
ethernet0.wakeOnPcktRcv = "FALSE"
ethernet0.addressType = "generated"

serial0.present = "TRUE"
serial0.fileType = "pipe"
serial0.yieldOnMsrRead = "TRUE"
serial0.startConnected = "TRUE"

pciBridge0.present = "TRUE"
pciBridge4.present = "TRUE"
pciBridge4.virtualDev = "pcieRootPort"
pciBridge4.functions = "8"

replay.supported = "FALSE"
hpet0.present = "TRUE"
vmci0.present = "FALSE"
mks.enable3d = "FALSE"
cleanShutdown = "TRUE"
softPowerOff = "FALSE"
usb.present = "FALSE"
ehci.present = "FALSE"
sound.present = "FALSE"
floppy0.present = "FALSE"
tools.syncTime = "FALSE"
`

func vmCreateVMXFile(c *VMConfig) error {
	file, err := os.OpenFile(c.VMXFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("Open file failed")
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.WriteString(vmx)

	str := "displayName = " + `"` + c.Name + `"` + "\n"
	writer.WriteString(str)

	str = "memsize = " + `"` + strconv.FormatInt(c.Memory, 10) + `"` + "\n"
	writer.WriteString(str)

	str = "numvcpus = " + `"` + strconv.Itoa(c.Cpus) + `"` + "\n"
	writer.WriteString(str)

	str = "serial0.fileName = " + `"` + c.sockPath() + `"`
	writer.WriteString(str)

	writer.Flush()
	return nil
}

func LoadConfig(name string) (*VMConfig, error) {
	dir := filepath.Join(util.ConfigDir(), "instances/vmw", name)
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

func vmList() ([]string, error) {
	cmd := exec.Command("vmrun", "list")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	vms := make([]string, 0)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		dir := filepath.Dir(line)
		vm := filepath.Base(dir)
		vms = append(vms, vm)
	}
	return vms, nil
}

func GetVMStatus(name, dir string) (string, error) {
	vms, err := vmList()
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
