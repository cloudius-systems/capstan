/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
	"net"
)

func HomePath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
	} else {
		return os.Getenv("HOME")
	}
}

func ID() string {
	return fmt.Sprintf("i%v", time.Now().Unix())
}

func CopyFile(src, dst string) *exec.Cmd {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", "copy", src, dst)
	} else {
		cmd = exec.Command("cp", src, dst)
	}
	return cmd
}

func SearchInstance(name string) (instanceName, instancePlatform string) {
	instanceName = ""
	instancePlatform = ""
	rootDir := filepath.Join(HomePath(), ".capstan", "instances")
	platforms, _ := ioutil.ReadDir(rootDir)
	for _, platform := range platforms {
		if !platform.IsDir() {
			continue
		}
		platformDir := filepath.Join(rootDir, platform.Name())
		instances, _ := ioutil.ReadDir(platformDir)
		for _, instance := range instances {
			if !instance.IsDir() {
				continue
			}
			if name != instance.Name() {
				continue
			}
			instanceName = instance.Name()
			instancePlatform = platform.Name()
			return
		}
	}
	return
}

func ConnectAndWait(path string) (net.Conn, error) {
	var conn net.Conn
	var err error
	for i := 0; i < 20; i++ {
		conn, err = Connect(path)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	return conn, err
}
