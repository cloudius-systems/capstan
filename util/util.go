/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func ConfigDir() string {
	return filepath.Join(HomePath(), ".capstan")
}

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

func CopyLocalFile(dst, src string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	// no need to check errors on read only file, we already got everything
	// we need from the filesystem, so nothing can go wrong now.
	defer s.Close()
	d, err := os.Create(dst)
	// Ensure the target file has the same mode as source
	d.Chmod(fi.Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

func SearchInstance(name string) (instanceName, instancePlatform string) {
	instanceName = ""
	instancePlatform = ""
	rootDir := filepath.Join(ConfigDir(), "instances")
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

			// Instance only exists if osv.config is present.
			if _, err := os.Stat(filepath.Join(platformDir, name, "osv.config")); os.IsNotExist(err) {
				// Search no more.
				return
			}

			instanceName = instance.Name()
			instancePlatform = platform.Name()
			return
		}
	}
	return
}

func ConnectAndWait(network, path string) (net.Conn, error) {
	var conn net.Conn
	var err error
	for i := 0; i < 20; i++ {
		conn, err = Connect(network, path)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	return conn, err
}

// RemoveOrphanedInstances removes directories of instances that were not persisted with --persist.
func RemoveOrphanedInstances(verbose bool) error {
	// TODO: Implement function InstancesPath()
	qemuDir := filepath.Join(ConfigDir(), "instances", "qemu")

	// Do nothing when instances/qemu folder does not exist.
	if _, err := os.Stat(qemuDir); os.IsNotExist(err) {
		return nil
	}

	instanceDirs, _ := ioutil.ReadDir(qemuDir)
	for _, instanceDir := range instanceDirs {
		if instanceDir.IsDir() {
			instanceDir := filepath.Join(qemuDir, instanceDir.Name())

			// Remove orphaned instance
			if _, err := os.Stat(filepath.Join(instanceDir, "osv.config")); os.IsNotExist(err) {
				if verbose {
					fmt.Println("Removing orphaned instance folder:", instanceDir)
				}

				if err = os.RemoveAll(instanceDir); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
