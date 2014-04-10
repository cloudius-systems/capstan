/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package qemu

import (
	"fmt"
	"gopkg.in/yaml.v1"
	"github.com/cloudius-systems/capstan/cpio"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/nbd"
	"github.com/cloudius-systems/capstan/util"
	"io"
	"io/ioutil"
	"bufio"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
	"path/filepath"
)

type VMConfig struct {
	Name	 string
	Image    string
	Verbose  bool
	Memory   int64
	Cpus     int
	NatRules []nat.Rule
	BackingFile bool
	InstanceDir string
	Monitor	string
	ConfigFile string
}

func UploadRPM(r *util.Repo, hypervisor string, image string, config *util.Config, verbose bool) error {
	file := r.ImagePath(hypervisor, image)
	vmconfig := &VMConfig{
		Image:    file,
		Verbose:  verbose,
		Memory:   64,
		NatRules: []nat.Rule{nat.Rule{GuestPort: "10000", HostPort: "10000"}},
		BackingFile: false,
	}
	qemu, err := LaunchVM(vmconfig)
	if err != nil {
		return err
	}
	defer qemu.Process.Kill()

	time.Sleep(1 * time.Second)

	conn, err := net.Dial("tcp", "localhost:10000")
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

	err = qemu.Wait()

	conn.Close()

	return err
}

func UploadFiles(r *util.Repo, hypervisor string, image string, config *util.Config, verbose bool) error {
	file := r.ImagePath(hypervisor, image)
	vmconfig := &VMConfig{
		Image:    file,
		Verbose:  verbose,
		Memory:   64,
		NatRules: []nat.Rule{nat.Rule{GuestPort: "10000", HostPort: "10000"}},
		BackingFile: false,
	}
	cmd, err := LaunchVM(vmconfig)
	if err != nil {
		return err
	}
	defer cmd.Process.Kill()

	time.Sleep(1 * time.Second)

	conn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		return err
	}

	for key, value := range config.Files {
		fi, err := os.Stat(value)
		if err != nil {
			return err
		}
		cpio.WritePadded(conn, cpio.ToWireFormat(key, cpio.C_ISREG, fi.Size()))
		b, err := ioutil.ReadFile(value)
		cpio.WritePadded(conn, b)
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
	time.Sleep(1 * time.Second)

	conn, err := net.Dial("tcp", "localhost:10809")
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

	data := append([]byte(args), make([]byte, 512-len(args))...)

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

func DeleteVM(name string) error {
	dir := filepath.Join(util.HomePath(), ".capstan/instances/qemu", name)
	c := &VMConfig{
		InstanceDir: dir,
		Monitor:  filepath.Join(dir, "osv.monitor"),
		Image:  filepath.Join(dir, "disk.qcow2"),
		ConfigFile:  filepath.Join(dir, "osv.config"),
	}
	cmd := exec.Command("rm", "-f", c.Image, " ", c.Monitor, " ", c.ConfigFile)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("rm failed: %s, %s", c.Image, c.Monitor);
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

func StopVM(name string) error {
	dir := filepath.Join(util.HomePath(), ".capstan/instances/qemu", name)
	c := &VMConfig{
		Monitor:  filepath.Join(dir, "osv.monitor"),
	}
	conn, err := net.Dial("unix", c.Monitor)
	if err != nil {
		fmt.Println("Failed to stop instance: %s", name)
		return err
	}
	writer := bufio.NewWriter(conn)

	cmd := `{ "execute": "qmp_capabilities"}`
	writer.WriteString(cmd)

	cmd = `{ "execute": "system_powerdown" }`
	writer.WriteString(cmd)

	cmd = `{ "execute": "quit" }`
	writer.WriteString(cmd)

	writer.Flush()

	return nil;
}

func LoadConfig(name string) (*VMConfig, error) {
	dir := filepath.Join(util.HomePath(), ".capstan/instances/qemu", name)
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

func LaunchVM(c *VMConfig, extra ...string) (*exec.Cmd, error) {
	if c.BackingFile {
		dir := c.InstanceDir
		err := os.MkdirAll(dir, 0775)
		if err != nil {
			fmt.Printf("mkdir failed: %s", dir);
			return nil, err
		}

		image, err := filepath.Abs(c.Image)
		if err != nil {
			fmt.Printf("Failed to open image %s\n", c.Image)
			return nil, err
		}
		backingFile := "backing_file=" + image
		newDisk := dir + "/disk.qcow2"

		if _, err := os.Stat(newDisk); os.IsNotExist(err) {
			cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-o", backingFile, newDisk)
			_, err = cmd.Output()
			if err != nil {
				fmt.Printf("qemu-img failed: %s", newDisk);
				return nil, err
			}
		}
		c.Image = newDisk
	}

	StoreConfig(c)

	args := append(c.vmArguments(), extra...)
	cmd := exec.Command("qemu-system-x86_64", args...)
	if c.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (c *VMConfig) vmArguments() []string {
	args := []string{"-display", "none","-m", strconv.FormatInt(c.Memory, 10), "-smp", strconv.Itoa(c.Cpus), "-device", "virtio-blk-pci,id=blk0,bootindex=0,drive=hd0", "-drive", "file=" + c.Image + ",if=none,id=hd0,aio=native,cache=none", "-netdev", "user,id=un0,net=192.168.122.0/24,host=192.168.122.1", "-device", "virtio-net-pci,netdev=un0", "-device", "virtio-rng-pci", "-chardev", "stdio,mux=on,id=stdio,signal=off", "-device", "isa-serial,chardev=stdio"}
	redirects := toQemuRedirects(c.NatRules)
	args = append(args, redirects...)
	monitor := fmt.Sprintf("socket,id=charmonitor,path=%s,server,nowait", c.Monitor)
	args = append(args, "-chardev", monitor, "-mon", "chardev=charmonitor,id=monitor,mode=control")
	if runtime.GOOS == "linux" {
		args = append(args, "-enable-kvm", "-cpu", "host,+x2apic")
	}
	return args
}

func toQemuRedirects(natRules []nat.Rule) []string {
	redirects := make([]string, 0)
	for _, portForward := range natRules {
		redirect := fmt.Sprintf("tcp:%s::%s", portForward.HostPort, portForward.GuestPort)
		redirects = append(redirects, "-redir", redirect)
	}
	return redirects
}
