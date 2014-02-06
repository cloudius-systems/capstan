/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cpio

import (
	"fmt"
	"net"
)

func WritePadded(c net.Conn, data []byte) {
	c.Write(data)
	partial := len(data)%4
	if partial != 0 {
		padding := make([]byte, 4-partial)
		c.Write(padding)
	}
}

func ToWireFormat(filename string, filesize int64) []byte {
	hdr := fmt.Sprintf("%s%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%s\u0000",
		"070701", // magic
		0, // inode
		0, // mode
		0, // uid
		0, // gid
		0, // nlink
		0, // mtime
		filesize, // filesize
		0, // devmajor
		0, // devminor
		0, // rdevmajor
		0, // rdevminor
		len(filename)+1, // namesize
		0, // check
		filename)
	return []byte(hdr)
}
