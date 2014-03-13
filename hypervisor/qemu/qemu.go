/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package qemu

import (
	"fmt"
	"github.com/cloudius-systems/capstan"
	"github.com/cloudius-systems/capstan/cpio"
	"github.com/cloudius-systems/capstan/nbd"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

type VMConfig struct {
	Image     string
	Verbose   bool
	Memory    int64
	Cpus      int
	Redirects []string
}

func UploadRPM(r *capstan.Repo, image string, config *capstan.Config, verbose bool) {
	file := r.ImagePath(image)
	vmconfig := &VMConfig{
		Image:     file,
		Verbose:   verbose,
		Memory:    64,
		Redirects: []string{"tcp:10000::10000"},
	}
	qemu, err := LaunchVM(vmconfig)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer qemu.Process.Kill()

	time.Sleep(1 * time.Second)

	conn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		fmt.Println(err)
		return
	}

	cmd := exec.Command("rpm2cpio", config.RpmBase.Filename())
	cmd.Stdout = conn
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cmd.Wait()

	qemu.Wait()
	conn.Close()
}

func UploadFiles(r *capstan.Repo, image string, config *capstan.Config, verbose bool) {
	file := r.ImagePath(image)
	vmconfig := &VMConfig{
		Image:     file,
		Verbose:   verbose,
		Memory:    64,
		Redirects: []string{"tcp:10000::10000"},
	}
	cmd, err := LaunchVM(vmconfig)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cmd.Process.Kill()

	time.Sleep(1 * time.Second)

	conn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		fmt.Println(err)
		return
	}

	for key, value := range config.Files {
		fi, err := os.Stat(value)
		if err != nil {
			fmt.Println(err)
			return
		}
		cpio.WritePadded(conn, cpio.ToWireFormat(key, cpio.C_ISREG, fi.Size()))
		b, err := ioutil.ReadFile(value)
		cpio.WritePadded(conn, b)
	}

	cpio.WritePadded(conn, cpio.ToWireFormat("TRAILER!!!", 0, 0))

	conn.Close()
	cmd.Wait()
}

func SetArgs(r *capstan.Repo, image string, args string) error {
	file := r.ImagePath(image)
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
	nbd.NbdHandshake(conn)

	data := append([]byte(args), make([]byte, 512-len(args))...)
	req := &nbd.NbdRequest{}
	req.Magic = nbd.NBD_REQUEST_MAGIC
	req.Type = nbd.NBD_CMD_WRITE
	req.Handle = 1 // running sequence number!
	req.From = 512
	req.Len = 512
	conn.Write(append(req.ToWireFormat(), data...))
	nbd.NbdRecv(conn)

	req = &nbd.NbdRequest{}
	req.Magic = nbd.NBD_REQUEST_MAGIC
	req.Type = nbd.NBD_CMD_FLUSH
	req.Handle = 2 // running sequence number!
	req.From = 0
	req.Len = 0
	conn.Write(req.ToWireFormat())
	nbd.NbdRecv(conn)

	req = &nbd.NbdRequest{}
	req.Magic = nbd.NBD_REQUEST_MAGIC
	req.Type = nbd.NBD_CMD_DISC
	req.Handle = 3 // running sequence number!
	req.From = 0
	req.Len = 0
	conn.Write(req.ToWireFormat())
	nbd.NbdRecv(conn)

	conn.Close()
	cmd.Wait()

	return nil
}

func LaunchVM(c *VMConfig, extra ...string) (*exec.Cmd, error) {
	args := append(c.vmArguments(), extra...)
	cmd := exec.Command("qemu-system-x86_64", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	if c.Verbose {
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
	}
	return cmd, nil
}

func (c *VMConfig) vmArguments() []string {
	redirects := make([]string, 0)
	for _, redirect := range c.Redirects {
		redirects = append(redirects, "-redir", redirect)
	}
	args := []string{"-vnc", ":1", "-gdb", "tcp::1234,server,nowait", "-m", strconv.FormatInt(c.Memory, 10), "-smp", strconv.Itoa(c.Cpus), "-device", "virtio-blk-pci,id=blk0,bootindex=0,drive=hd0", "-drive", "file=" + c.Image + ",if=none,id=hd0,aio=native,cache=none", "-netdev", "user,id=un0,net=192.168.122.0/24,host=192.168.122.1", "-device", "virtio-net-pci,netdev=un0", "-device", "virtio-rng-pci", "-chardev", "stdio,mux=on,id=stdio,signal=off", "-mon", "chardev=stdio,mode=readline,default", "-device", "isa-serial,chardev=stdio"}
	args = append(args, redirects...)
	if runtime.GOOS == "linux" {
		args = append(args, "-enable-kvm", "-cpu", "host,+x2apic")
	}
	return args
}
