// +build linux darwin

/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"net"
)

func Connect(path string) (net.Conn, error) {
	return net.Dial("unix", path)
}
