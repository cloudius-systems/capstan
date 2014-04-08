/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"os/exec"
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
