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
	"strings"
	"testing"
)

func runCapstan(command []string, root string) *exec.Cmd {
	c := exec.Command("capstan", command...)
	c.Env = append(os.Environ(), fmt.Sprintf("CAPSTAN_ROOT=%s", root))
	return c
}

func TestCommandErrors(t *testing.T) {
	root, err := ioutil.TempDir("", "capstan-root")
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	defer os.RemoveAll(root)

	m := map[string]string{
		"build foo":  "open Capstanfile: no such file or directory\n",
		"build":      "usage: capstan build [image-name]\n",
		"pull":       "usage: capstan pull [image-name]\n",
		"import":     "usage: capstan import [image-name] [image-file]\n",
		"import foo": "usage: capstan import [image-name] [image-file]\n",
		"rmi":        "usage: capstan rmi [image-name]\n",
		"run foo":    "foo: no such image\n",
		"run":        "No Capstanfile found, unable to run.\n",
	}
	for key, value := range m {
		cmd := runCapstan(strings.Fields(key), root)
		out, err := cmd.Output()
		if err != nil {
			t.Errorf("capstan: %v", err)
		}
		if g, e := string(out), value; g != e {
			t.Errorf("capstan: want %q, got %q", e, g)
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
	if g, e := string(out), "Importing example...\n"; g != e {
		t.Errorf("capstan: want %q, got %q", e, g)
	}

	cmd = runCapstan([]string{"images"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	outLines := strings.Split(string(out), "\n")
	if g, e := outLines[1], "example"; g != e {
		t.Errorf("capstan: want %q, got %q", e, g)

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
	if g, e := string(out), "Importing example1...\n"; g != e {
		t.Errorf("capstan: want %q, got %q", e, g)
	}

	cmd = runCapstan([]string{"import", "example2", "example.qcow2"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	if g, e := string(out), "Importing example2...\n"; g != e {
		t.Errorf("capstan: want %q, got %q", e, g)
	}

	cmd = runCapstan([]string{"images"}, root)
	out, err = cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	outLines := strings.Split(string(out), "\n")
	if g, e := outLines[1]+"\n"+outLines[2], "example1\nexample2"; g != e {
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
	if g, e := outLines[1], "example2"; g != e {
		t.Errorf("capstan: want %q, got %q", e, g)

	}
}
