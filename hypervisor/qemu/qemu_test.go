/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package qemu

import (
	"testing"

	. "github.com/mikelangelo-project/capstan/testing"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type suite struct{}

var _ = Suite(&suite{})

func (*suite) TestVersionParsing(c *C) {
	m := []struct {
		in  string
		out *Version
	}{
		{"QEMU emulator version 1.0 (qemu-kvm-1.0), Copyright (c) 2003-2008 Fabrice Bellard", &Version{Major: 1, Minor: 0, Patch: 0}},
		{"QEMU emulator version 1.6.2, Copyright (c) 2003-2008 Fabrice Bellard", &Version{Major: 1, Minor: 6, Patch: 2}},
		{"QEMU PC emulator version 0.12.1 (qemu-kvm-0.12.1.2), Copyright (c) 2003-2008 Fabrice Bellard", &Version{Major: 0, Minor: 12, Patch: 1}},
	}
	for i, args := range m {
		c.Logf("CASE #%d", i)

		// This is what we're testing here.
		version, err := ParseVersion(args.in)

		// Expectations.
		c.Check(err, IsNil)
		c.Check(version, DeepEquals, args.out)
	}
}

func (s *suite) TestVmArguments(c *C) {
	m := []struct {
		comment  string
		config   VMConfig
		expected []string
	}{
		{
			"basic",
			VMConfig{},
			[]string{
				"-nographic",
				"-m", "0",
				"-smp", "0",
				"-device", "virtio-blk-pci,id=blk0,bootindex=0,drive=hd0",
				"-drive", "file=,if=none,id=hd0,aio=threads,cache=unsafe",
				"-device", "virtio-rng-pci",
				"-chardev", "stdio,mux=on,id=stdio,signal=off",
				"-device", "isa-serial,chardev=stdio",
				"-netdev", "user,id=un0,net=192.168.122.0/24,host=192.168.122.1",
				"-device", "virtio-net-pci,netdev=un0",
				"-chardev", "socket,id=charmonitor,path=,server,nowait",
				"-mon", "chardev=charmonitor,id=monitor,mode=control",
			},
		},
		// Volumes.
		{
			"single volume",
			VMConfig{
				Volumes: []string{"/path/vol1.img"},
			},
			[]string{
				"-drive", "file=/path/vol1.img,if=none,id=hd1,aio=native,cache=none,format=raw",
				"-device", "virtio-blk-pci,id=blk1,bootindex=1,drive=hd1",
			},
		},
		{
			"single volume with metadata",
			VMConfig{
				Volumes: []string{"/path/vol1.img:format=qcow2:aio=threads:cache=writethrough"},
			},
			[]string{
				"-drive", "file=/path/vol1.img,if=none,id=hd1,aio=threads,cache=writethrough,format=qcow2",
				"-device", "virtio-blk-pci,id=blk1,bootindex=1,drive=hd1",
			},
		},
		{
			"two volumes",
			VMConfig{
				Volumes: []string{"/path/vol1.img", "/path/vol2.img"},
			},
			[]string{
				"-drive", "file=/path/vol1.img,if=none,id=hd1,aio=native,cache=none,format=raw",
				"-device", "virtio-blk-pci,id=blk1,bootindex=1,drive=hd1",
				"-drive", "file=/path/vol2.img,if=none,id=hd2,aio=native,cache=none,format=raw",
				"-device", "virtio-blk-pci,id=blk2,bootindex=2,drive=hd2",
			},
		},
		{
			"two volumes, one with metadata",
			VMConfig{
				Volumes: []string{"/path/vol1.img:format=qcow2", "/path/vol2.img"},
			},
			[]string{
				"-drive", "file=/path/vol1.img,if=none,id=hd1,aio=native,cache=none,format=qcow2",
				"-device", "virtio-blk-pci,id=blk1,bootindex=1,drive=hd1",
				"-drive", "file=/path/vol2.img,if=none,id=hd2,aio=native,cache=none,format=raw",
				"-device", "virtio-blk-pci,id=blk2,bootindex=2,drive=hd2",
			},
		},
		// AioType.
		{
			"aio type native",
			VMConfig{
				AioType: "native",
			},
			[]string{
				"-device", "virtio-blk-pci,id=blk0,bootindex=0,drive=hd0",
				"-drive", "file=,if=none,id=hd0,aio=native,cache=unsafe",
			},
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		qemuVersion := &Version{Major: 2, Minor: 5, Patch: 0}
		config := s.setDefaultAttributes(args.config, c)

		// This is what we're testing here.
		qemuArgs, err := config.vmArguments(qemuVersion)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(qemuArgs, ContainsArray, args.expected)
	}
}

func (*suite) setDefaultAttributes(conf VMConfig, c *C) VMConfig {
	conf.DisableKvm = true
	if conf.Networking == "" {
		conf.Networking = "nat"
	}
	if conf.AioType == "" {
		conf.AioType = "threads"
	}

	return conf
}
