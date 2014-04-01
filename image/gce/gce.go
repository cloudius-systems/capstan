/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package gce

import (
	"encoding/binary"
	"os"
)

const (
	GZ_MAGIC1 = 0x1F
	GZ_MAGIC2 = 0x8B

)
type Header struct {
	MagicNumber1        uint8
	MagicNumber2        uint8
}

func Probe(f *os.File) bool {
	header, err := readHeader(f)
	if err != nil {
		return false
	}
	return (header.MagicNumber1 == GZ_MAGIC1) && (header.MagicNumber2 == GZ_MAGIC2)
}

func readHeader(f *os.File) (*Header, error) {
	var header Header
	err := binary.Read(f, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	return &header, nil
}
