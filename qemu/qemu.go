/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package qemu

import (
	"fmt"
	"github.com/cloudius-systems/capstan/cpio"
	"github.com/cloudius-systems/capstan/nbd"
	"github.com/cloudius-systems/capstan"
	"github.com/vaughan0/go-ini"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func BuildImage(r *capstan.Repo, image string) {
	inifile, _ := ini.LoadFile("Capstanfile")
	cmdline, ok := inifile.Get("config", "cmdline")
	if !ok {
		panic("'cmdline' variable missing from 'config' section")
	}
	base, ok := inifile.Get("config", "base")
	if !ok {
		panic("'base' variable missing from 'config' section")
	}
	if _, err := os.Stat(r.ImagePath(base)); os.IsNotExist(err) {
		fmt.Printf("%s: no such base image\n", base)
		return
	}
	for _, value := range inifile["manifest"] {
		if _, err := os.Stat(value); os.IsNotExist(err) {
			fmt.Printf("%s: no such file or directory\n", value)
			return
		}
	}
	fmt.Printf("Building %s...\n", image)
	err := os.MkdirAll(filepath.Dir(r.ImagePath(image)), 0777)
	if err != nil {
		panic(err)
	}
	cmd := exec.Command("cp", r.ImagePath(base), r.ImagePath(image))
	_, err = cmd.Output()
	if err != nil {
		println(err.Error())
		return
	}
	SetArgs(r, image, "/tools/cpiod.so")
	UploadFiles(r, image, inifile)
	SetArgs(r, image, cmdline)
}

func UploadFiles(r *capstan.Repo, image string, inifile ini.File) {
	cmd := LaunchVM(r, image, "-redir", "tcp:10000::10000")
	defer cmd.Process.Kill()

	time.Sleep(1 * time.Second)

	conn, err := net.Dial("tcp", "localhost:10000")
	if err != nil {
		fmt.Println(err)
		return
	}

	for key, value := range inifile["manifest"] {
		fi, err := os.Stat(value)
		if err != nil {
			fmt.Println(err)
			return
		}
		cpio.WritePadded(conn, cpio.ToWireFormat(key, fi.Size()))
		b, err := ioutil.ReadFile(value)
		cpio.WritePadded(conn, b)
	}

	cpio.WritePadded(conn, cpio.ToWireFormat("TRAILER!!!", 0))

	conn.Close()
	cmd.Wait()
}

func SetArgs(r *capstan.Repo, image string, args string) {
	file := r.ImagePath(image)
	cmd := exec.Command("qemu-nbd", file)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
	}
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
		return
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	time.Sleep(1 * time.Second)
	conn, err := net.Dial("tcp", "localhost:10809")
	if err != nil {
		fmt.Println(err)
		return
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
}

func LaunchVM(r *capstan.Repo, image string, extra ...string) *exec.Cmd {
	file := r.ImagePath(image)
	args := append([]string{"-vnc", ":1", "-gdb", "tcp::1234,server,nowait", "-m", "2G", "-smp", "4", "-device", "virtio-blk-pci,id=blk0,bootindex=0,drive=hd0,scsi=off", "-drive", "file=" + file + ",if=none,id=hd0,aio=native,cache=none", "-netdev", "user,id=un0,net=192.168.122.0/24,host=192.168.122.1", "-redir", "tcp:8080::8080", "-redir", "tcp:2222::22", "-device", "virtio-net-pci,netdev=un0", "-device", "virtio-rng-pci", "-enable-kvm", "-cpu", "host,+x2apic", "-chardev", "stdio,mux=on,id=stdio,signal=off", "-mon", "chardev=stdio,mode=readline,default", "-device", "isa-serial,chardev=stdio"}, extra...)
	cmd := exec.Command("qemu-system-x86_64", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
	}
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	return cmd
}
