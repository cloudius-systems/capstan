/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 * Modifications copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func runCapstan(command []string, root string) *exec.Cmd {
	c := exec.Command("capstan", command...)
	c.Env = append(os.Environ(), fmt.Sprintf("CAPSTAN_ROOT=%s", root))
	return c
}

func TestCommandErrorCodes(t *testing.T) {
	root, err := ioutil.TempDir("", "capstan-root")
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	defer os.RemoveAll(root)

	m := []struct {
		cmd     string
		errCode int
		errMsg  string
	}{
		{"build foo", 65, "open Capstanfile: no such file or directory\n"},
		{"build", 64, "usage: capstan build [image-name]\n"},
		{"pull", 64, "usage: capstan pull [image-name]\n"},
		{"import", 64, "usage: capstan import [image-name] [image-file]\n"},
		{"import foo", 64, "usage: capstan import [image-name] [image-file]\n"},
		{"rmi", 64, "usage: capstan rmi [image-name]\n"},
		{"run foo", 65, "Command line will be set to default boot\nfoo: no such image at: " + root + "/repository/foo/foo.qemu\n"},
		{"run", 65, "Missing Capstanfile or package metadata\n"},
		{"package help compose", 0, "capstan package compose - composes the package and all its dependencies into OSv imag"},
	}
	for _, args := range m {
		cmd := runCapstan(strings.Fields(args.cmd), root)
		out, err := cmd.CombinedOutput()
		if (args.errCode == 0) && (err != nil) ||
			(args.errCode != 0) && (err == nil) ||
			(err != nil) && !strings.Contains(err.Error(), fmt.Sprintf("%d", args.errCode)) {
			t.Errorf("capstan %s: %v", args.cmd, err)
		}
		if g, e := string(out), args.errMsg; !strings.Contains(g, e) {
			t.Errorf("capstan %s: want %q, got %q", args.cmd, e, g)
		}
	}
}

func TestImportCommand(t *testing.T) {
	root, err := ioutil.TempDir("", "capstan-root")
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	defer os.RemoveAll(root)
	defer os.Remove("example.qcow2")

	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "example.qcow2", "128M")
	out, err := cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}

	cmd = runCapstan([]string{"import", "example", "example.qcow2"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	expectedMessage := fmt.Sprintf("Importing example...\nImporting into %s/repository/example/example.qemu\n", root)
	if g := string(out); g != expectedMessage {
		t.Errorf("capstan: want %q, got %q", expectedMessage, g)
	}

	cmd = runCapstan([]string{"images"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	outLines := strings.Split(string(out), "\n")
	if g, e := outLines[1], "example .*\n"; regexp.MustCompile(e).MatchString(g) {
		t.Errorf("capstan: want prefix %q, got %q", e, g)
	}
}

func TestRmiCommand(t *testing.T) {
	root, err := ioutil.TempDir("", "capstan-root")
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	defer os.RemoveAll(root)
	defer os.Remove("example.qcow2")

	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "example.qcow2", "128M")
	out, err := cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}

	cmd = runCapstan([]string{"import", "example1", "example.qcow2"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	e := fmt.Sprintf("Importing example1...\nImporting into %s/repository/example1/example1.qemu\n", root)
	if g := string(out); g != e {
		t.Errorf("capstan: want %q, got %q", e, g)
	}

	cmd = runCapstan([]string{"import", "example2", "example.qcow2"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	e = fmt.Sprintf("Importing example2...\nImporting into %s/repository/example2/example2.qemu\n", root)
	if g := string(out); g != e {
		t.Errorf("capstan: want %q, got %q", e, g)
	}

	cmd = runCapstan([]string{"images"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	outLines := strings.Split(string(out), "\n")
	if g, e := outLines[1]+"\n"+outLines[2], "example1 .*\nexample2 .*\n"; regexp.MustCompile(e).MatchString(g) {
		t.Errorf("capstan: want %q, got %q", e, g)
	}

	cmd = runCapstan([]string{"rmi", "example1"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	if g, e := string(out), "Removing example1...\n"; g != e {
		t.Errorf("capstan: want %q, got %q", e, g)

	}

	cmd = runCapstan([]string{"images"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	outLines = strings.Split(string(out), "\n")
	if g, e := outLines[1], "example2 .*\n"; regexp.MustCompile(e).MatchString(g) {
		t.Errorf("capstan: want %q, got %q", e, g)
	}
}
