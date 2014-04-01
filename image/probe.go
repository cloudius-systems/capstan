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
	"github.com/cloudius-systems/capstan/image/vmdk"
	"github.com/cloudius-systems/capstan/image/gce"
	"os"
)

type ImageFormat int

const (
	QCOW2 ImageFormat = iota
	VDI
	VMDK
	GCE
	Unknown
)

func Probe(path string) (ImageFormat, error) {
	f, err := os.Open(path)
	if err != nil {
		return Unknown, err
	}
	defer f.Close()
	f.Seek(0, os.SEEK_SET)
	if qcow2.Probe(f) {
		return QCOW2, nil
	}
	f.Seek(0, os.SEEK_SET)
	if vdi.Probe(f) {
		return VDI, nil
	}
	f.Seek(0, os.SEEK_SET)
	if vmdk.Probe(f) {
		return VMDK, nil
	}
	f.Seek(0, os.SEEK_SET)
	if gce.Probe(f) {
		return GCE, nil
	}
	return Unknown, nil
}
