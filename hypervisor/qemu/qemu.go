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
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func BuildImage(r *capstan.Repo, image string, verbose bool) error {
	config, err := capstan.ReadConfig("Capstanfile")
	if err != nil {
		return err
	}
	fmt.Printf("Building %s...\n", image)
	err = os.MkdirAll(filepath.Dir(r.ImagePath(image)), 0777)
	if err != nil {
		return err
	}
	if config.RpmBase != nil {
		config.RpmBase.Download()
	}
	if config.Build != "" {
		args := strings.Fields(config.Build)
		cmd := exec.Command(args[0], args[1:]...)
		_, err = cmd.Output()
		if err != nil {
			return err
		}
	}
	err = config.Check(r)
	if err != nil {
		return err
	}
	cmd := exec.Command("cp", r.ImagePath(config.Base), r.ImagePath(image))
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	err = SetArgs(r, image, "/tools/cpiod.so")
	if err != nil {
		return err
	}
	if config.RpmBase != nil {
		UploadRPM(r, image, config, verbose)
	}
	UploadFiles(r, image, config, verbose)
	err = SetArgs(r, image, config.Cmdline)
	if err != nil {
		return err
	}
	return nil
}

func UploadRPM(r *capstan.Repo, image string, config *capstan.Config, verbose bool) {
	file := r.ImagePath(image)
	qemu := LaunchVM(verbose, file, "-redir", "tcp:10000::10000")
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
	cmd := LaunchVM(verbose, file, "-redir", "tcp:10000::10000")
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

func LaunchVM(verbose bool, file string, extra ...string) *exec.Cmd {
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
	if verbose {
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
	}
	return cmd
}
