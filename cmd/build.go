/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/cheggaaa/pb"
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
	if err := os.MkdirAll(filepath.Dir(r.ImagePath(hypervisor, image)), 0777); err != nil {
		return err
	}
	if err := checkConfig(config, r, hypervisor); err != nil {
		return err
	}
	if config.RpmBase != nil {
		config.RpmBase.Download()
	}
	fmt.Printf("Building %s...\n", image)
	if config.Build != "" {
		args := strings.Fields(config.Build)
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return err
		}
	}
	cmd := util.CopyFile(r.ImagePath(hypervisor, config.Base), r.ImagePath(hypervisor, image))
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	if err := SetArgs(r, hypervisor, image, "/tools/cpiod.so"); err != nil {
		return err
	}
	if config.RpmBase != nil {
		if err := UploadRPM(r, hypervisor, image, config, verbose, mem); err != nil {
			return err
		}
	}
	if err := UploadFiles(r, hypervisor, image, config, verbose, mem); err != nil {
		return err
	}
	return SetArgs(r, hypervisor, image, config.Cmdline)
}

func checkConfig(config *util.Config, r *util.Repo, hypervisor string) error {
	if _, err := os.Stat(r.ImagePath(hypervisor, config.Base)); os.IsNotExist(err) {
		if err := Pull(r, hypervisor, config.Base); err != nil {
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

func IsReg(m os.FileMode) bool {
	nonreg := os.ModeDir | os.ModeSymlink | os.ModeDevice | os.ModeSocket | os.ModeCharDevice
	return (m & nonreg) == 0
}

func copyFile(conn net.Conn, src string, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		cpio.WritePadded(conn, cpio.ToWireFormat(dst, cpio.C_ISDIR, 0))
		return nil
	}

	if !IsReg(fi.Mode()) {
		fmt.Println("skipping non-file path " + src)
		return nil
	} else {
		contents, err := ioutil.ReadFile(src)
		if err != nil {
			return nil
		}
		cpio.WritePadded(conn, cpio.ToWireFormat(dst, cpio.C_ISREG, fi.Size()))
		cpio.WritePadded(conn, contents)
	}

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
	cmd, err := qemu.VMCommand(vmconfig)
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Process.Kill()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		if verbose {
			fmt.Println(text)
		}
		if text == "Waiting for connection from host..." {
			break
		}
	}
	if verbose {
		go io.Copy(os.Stdout, stdout)
	}
	conn, err := util.ConnectAndWait("tcp", "localhost:10000")
	if err != nil {
		return err
	}

	rootfsFiles := make(map[string]string)
	if _, err = os.Stat(config.Rootfs); !os.IsNotExist(err) {
		err = filepath.Walk(config.Rootfs, func(src string, info os.FileInfo, _ error) error {
			dst := strings.Replace(src, config.Rootfs, "", 1)
			rootfsFiles[dst] = src
			return nil
		})
	}

	fmt.Println("Uploading files...")

	bar := pb.New(len(rootfsFiles) + len(config.Files))

	if !verbose {
		bar.Start()
	}

	for dst, src := range rootfsFiles {
		err = copyFile(conn, src, dst)
		if verbose {
			fmt.Println(src + "  --> " + dst)
		} else {
			bar.Increment()
		}
		if err != nil {
			return err
		}
	}

	for dst, src := range config.Files {
		err = copyFile(conn, src, dst)
		if verbose {
			fmt.Println(src + "  --> " + dst)
		} else {
			bar.Increment()
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
