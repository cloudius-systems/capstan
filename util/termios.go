// +build linux darwin

/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"os/exec"
)

func RawTerm() error {
	cmd := exec.Command("stty", "raw")
	return cmd.Run()
}

func ResetTerm() {
	cmd := exec.Command("stty", "cooked")
	cmd.Run()
}
