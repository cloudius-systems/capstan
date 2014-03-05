/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package image

import (
	"github.com/cloudius-systems/capstan/image/qcow2"
	"github.com/cloudius-systems/capstan/image/vdi"
	"os"
)

type ImageFormat int

const (
	QCOW2 ImageFormat = iota
	VDI
	Unknown
)

func Probe(f *os.File) ImageFormat {
	f.Seek(0, os.SEEK_SET)
	if qcow2.Probe(f) {
		return QCOW2
	}
	f.Seek(0, os.SEEK_SET)
	if vdi.Probe(f) {
		return VDI
	}
	return Unknown
}
