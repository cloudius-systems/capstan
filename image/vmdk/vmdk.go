/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package vmdk

import (
	"encoding/binary"
	"os"
)

const (
	VMDK_MAGIC = 0x564d444b
)

type SectorType uint64
type Bool uint8

type Header struct {
	MagicNumber        uint32
	Version            uint32
	Flags              uint32
	Capacity           SectorType
	GrainSize          SectorType
	DescriptorOffset   SectorType
	DescriptorSize     SectorType
	NumGTEsPerGT       uint32
	RgdOffset          SectorType
	GdOffset           SectorType
	OverHead           SectorType
	UncleanShutdown    Bool
	SingleEndLineChar  byte
	NonEndLineChar     byte
	DoubleEndLineChar1 byte
	DoubleEndLineChar2 byte
	CompressAlgorithm  uint16
	Pad                [433]uint8
}

func Probe(f *os.File) bool {
	header, err := readHeader(f)
	if err != nil {
		return false
	}
	return header.MagicNumber == VMDK_MAGIC
}

func readHeader(f *os.File) (*Header, error) {
	var header Header
	err := binary.Read(f, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	return &header, nil
}
