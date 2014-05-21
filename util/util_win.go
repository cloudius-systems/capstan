// +build windows

/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"github.com/natefinch/npipe"
	"net"
)

func Connect(network, path string) (net.Conn, error) {
	return npipe.Dial(path)
}
