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
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/util"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
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
	dir := c.InstanceDir
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		fmt.Printf("mkdir failed: %s", dir)
		return nil, err
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", "copy", c.OriginalVMDK, c.Image)
	} else {
		cmd = exec.Command("cp", c.OriginalVMDK, c.Image)
	}
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
	cmd, err = vmxRun("-T", "ws", "start", c.VMXFile, "nogui")
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	for i:= 0; i < 5; i++ {
		conn, err = Connect(c.sockPath())
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		fmt.Println("err socket")
		return nil, err
	}

	done := make(chan bool)
	go io.Copy(conn, os.Stdin)
	go doRead(done, conn)

	// Wait until the serial connection is disconnected
	<-done

	return cmd, nil
}

func DeleteVM(c *VMConfig) error {
	cmd, err := vmxRun("-T", "ws", "deleteVM", c.VMXFile)
	if err != nil {
		fmt.Printf("Failed to delete VM", c.VMXFile)
		return err
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Failed to delete VM", c.VMXFile)
		return err
	}
	return nil
}

func StopVM(name string) error {
	dir := filepath.Join(util.HomePath(), ".capstan/instances/vmw", name)
	c := &VMConfig{
		VMXFile:  filepath.Join(dir, "osv.vmx"),
	}
	cmd, err := vmxRun("-T", "ws", "stop", c.VMXFile)
	if err != nil {
		fmt.Printf("Failed to stop VM", c.VMXFile)
		return err
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Failed to stop VM", c.VMXFile)
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

var vmx string =
`#!/usr/bin/vmware
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
