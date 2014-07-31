/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"errors"
	"fmt"
	"github.com/cloudius-systems/capstan/cpio"
	"github.com/cloudius-systems/capstan/hypervisor/qemu"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/nbd"
	"github.com/cloudius-systems/capstan/util"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Build(r *util.Repo, hypervisor string, image string, verbose bool, mem string) error {
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
	err = SetArgs(r, hypervisor, image, "/tools/cpiod.so")
	if err != nil {
		return err
	}
	if config.RpmBase != nil {
		err = UploadRPM(r, hypervisor, image, config, verbose, mem)
		if err != nil {
			return err
		}
	}
	err = UploadFiles(r, hypervisor, image, config, verbose, mem)
	if err != nil {
		return err
	}
	err = SetArgs(r, hypervisor, image, config.Cmdline)
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

func UploadRPM(r *util.Repo, hypervisor string, image string, config *util.Config, verbose bool, mem string) error {
	file := r.ImagePath(hypervisor, image)
	size, err := util.ParseMemSize(mem)
	if err != nil {
		return err
	}
	vmconfig := &qemu.VMConfig{
		Image:       file,
		Verbose:     verbose,
		Memory:      size,
		Networking:  "nat",
		NatRules:    []nat.Rule{nat.Rule{GuestPort: "10000", HostPort: "10000"}},
		BackingFile: false,
	}
	vm, err := qemu.LaunchVM(vmconfig)
	if err != nil {
		return err
	}
	defer vm.Process.Kill()

	conn, err := util.ConnectAndWait("tcp", "localhost:10000")
	if err != nil {
		return err
	}

	cmd := exec.Command("rpm2cpio", config.RpmBase.Filename())
	cmd.Stdout = conn
	err = cmd.Start()
	if err != nil {
		return err
	}
	defer cmd.Wait()

	err = vm.Wait()

	conn.Close()

	return err
}

func copyFile(conn net.Conn, src string, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	cpio.WritePadded(conn, cpio.ToWireFormat(dst, cpio.C_ISREG, fi.Size()))
	b, err := ioutil.ReadFile(src)
	cpio.WritePadded(conn, b)
	return nil
}

func UploadFiles(r *util.Repo, hypervisor string, image string, config *util.Config, verbose bool, mem string) error {
	file := r.ImagePath(hypervisor, image)
	size, err := util.ParseMemSize(mem)
	if err != nil {
		return err
	}
	vmconfig := &qemu.VMConfig{
		Image:       file,
		Verbose:     verbose,
		Memory:      size,
		Networking:  "nat",
		NatRules:    []nat.Rule{nat.Rule{GuestPort: "10000", HostPort: "10000"}},
		BackingFile: false,
	}
	cmd, err := qemu.LaunchVM(vmconfig)
	if err != nil {
		return err
	}
	defer cmd.Process.Kill()

	conn, err := util.ConnectAndWait("tcp", "localhost:10000")
	if err != nil {
		return err
	}

	if _, err = os.Stat(config.Rootfs); !os.IsNotExist(err) {
		err = filepath.Walk(config.Rootfs, func(src string, info os.FileInfo, _ error) error {
			if info.IsDir() {
				return nil
			}
			dst := strings.Replace(src, config.Rootfs, "", -1)
			if verbose {
				fmt.Println(src + "  --> " + dst)
			}
			return copyFile(conn, src, dst)
		})
	}

	for dst, src := range config.Files {
		err = copyFile(conn, src, dst)
		if verbose {
			fmt.Println(src + "  --> " + dst)
		}
		if err != nil {
			return err
		}
	}

	cpio.WritePadded(conn, cpio.ToWireFormat("TRAILER!!!", 0, 0))

	conn.Close()
	return cmd.Wait()
}

func SetArgs(r *util.Repo, hypervisor, image string, args string) error {
	file := r.ImagePath(hypervisor, image)
	cmd := exec.Command("qemu-nbd", file)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()
	if err != nil {
		return err
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	conn, err := util.ConnectAndWait("tcp", "localhost:10809")
	if err != nil {
		return err
	}

	session := &nbd.NbdSession{
		Conn:   conn,
		Handle: 0,
	}
	if err := session.Handshake(); err != nil {
		return err
	}

	padding := 512 - (len(args) % 512)

	data := append([]byte(args), make([]byte, padding)...)

	if err := session.Write(512, data); err != nil {
		return err
	}
	if err := session.Flush(); err != nil {
		return err
	}
	if err := session.Disconnect(); err != nil {
		return err
	}
	conn.Close()
	cmd.Wait()

	return nil
}
