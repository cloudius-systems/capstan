package nbd

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	NBD_REQUEST_MAGIC = 0x25609513
	NBD_REPLY_MAGIC   = 0x67446698
)

const (
	NBD_CMD_READ  = 0
	NBD_CMD_WRITE = 1
	NBD_CMD_DISC  = 2
	NBD_CMD_FLUSH = 3
	NBD_CMD_TRIM  = 4
)

type NbdRequest struct {
	Magic  uint32
	Type   uint32
	Handle uint64
	From   uint64
	Len    uint32
}

type NbdReply struct {
	Magic  uint32
	Error  uint32
	Handle uint64
}

func (msg *NbdRequest) ToWireFormat() []byte {
	endian := binary.BigEndian
	b := make([]byte, 4+4+8+8+4)
	endian.PutUint32(b[0:4], msg.Magic)
	endian.PutUint32(b[4:8], msg.Type)
	endian.PutUint64(b[8:16], msg.Handle)
	endian.PutUint64(b[16:24], msg.From)
	endian.PutUint32(b[24:28], msg.Len)
	return b
}

func NbdHandshake(c net.Conn) {
	nbd_magic := make([]byte, len("NBDMAGIC"))
	c.Read(nbd_magic)
	if string(nbd_magic) != "NBDMAGIC" {
		fmt.Println("NBD magic missing!")
	}
	c.Read(make([]byte, 8+8+4))
	c.Read(make([]byte, 124))
}

func NbdRecv(c net.Conn) {
	c.Read(make([]byte, 4+4+8))
}
