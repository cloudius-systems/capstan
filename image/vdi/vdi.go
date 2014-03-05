package vdi

import (
	"encoding/binary"
	"os"
)

const (
	VDI_SIGNATURE = 0xbeda107f
)

type VdiHeader struct {
	Text            [0x40]byte
	Signature       uint32
	Version         uint32
	HeaderSize      uint32
	ImageType       uint32
	ImageFlags      uint32
	Description     [256]byte
	OffsetBmap      uint32
	OffsetData      uint32
	Cylinders       uint32
	Heads           uint32
	Sectors         uint32
	SectorSize      uint32
	Unused1         uint32
	DiskSize        uint64
	BlockSize       uint32
	BlockExtra      uint32
	BlocksInImage   uint32
	BlocksAllocated uint32
	UuidImage       [16]byte
	UuidLastSnap    [16]byte
	UuidLink        [16]byte
	UuidParent      [16]byte
	Unused2         [7]uint64
}

func VdiReadHeader(f *os.File) (*VdiHeader, error) {
	var header VdiHeader
	err := binary.Read(f, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func Probe(f *os.File) bool {
	header, err := VdiReadHeader(f)
	if err != nil {
		return false
	}
	return header.Signature == VDI_SIGNATURE
}
