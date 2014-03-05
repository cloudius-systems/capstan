package qcow2

import (
	"encoding/binary"
	"os"
)

const (
	QCOW2_MAGIC = ('Q' << 24) | ('F' << 16) | ('I' << 8) | 0xfb
)

type VdiHeader struct {
	Magic                 uint32
	Version               uint32
	BackingFileOffset     uint64
	BackingFileSize       uint32
	ClusterBits           uint32
	Size                  uint64
	CryptMethod           uint32
	L1Size                uint32
	L1TableOffset         uint64
	RefcountTableOffset   uint64
	RefcountTableClusters uint32
	NbSnapshots           uint32
	SnapshotsOffset       uint64
}

func VdiReadHeader(f *os.File) (*VdiHeader, error) {
	var header VdiHeader
	err := binary.Read(f, binary.BigEndian, &header)
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
	return header.Magic == QCOW2_MAGIC
}
