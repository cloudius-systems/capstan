/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

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

const (
	NBD_FLAG_HAS_FLAGS  = (1 << 0)
	NBD_FLAG_SEND_FLUSH = (1 << 2)
)

type NbdSession struct {
	Conn   net.Conn
	Handle uint64
	Size   uint64
	Flags  uint32
}

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

func (session *NbdSession) Handshake() error {
	nbd_magic := make([]byte, len("NBDMAGIC"))
	session.Conn.Read(nbd_magic)
	if string(nbd_magic) != "NBDMAGIC" {
		return fmt.Errorf("NBD magic missing!")
	}
	buf := make([]byte, 8+8+4)
	session.Conn.Read(buf)
	magic := binary.BigEndian.Uint64(buf)
	if magic != 0x00420281861253 {
		return fmt.Errorf("Bad magic: %x", magic)
	}
	session.Size = binary.BigEndian.Uint64(buf)
	session.Flags = binary.BigEndian.Uint32(buf)
	session.Conn.Read(make([]byte, 124))
	session.Handle += 1
	return nil
}

func (session *NbdSession) Write(from uint64, data []byte) error {
	req := &NbdRequest{
		Magic:  NBD_REQUEST_MAGIC,
		Type:   NBD_CMD_WRITE,
		Handle: session.Handle,
		From:   512,
		Len:    uint32(len(data)),
	}
	_, err := session.Conn.Write(append(req.ToWireFormat(), data...))
	if err != nil {
		return err
	}
	return session.Recv()
}

func (session *NbdSession) needFlush() bool {
	return (session.Flags&NBD_FLAG_HAS_FLAGS == NBD_FLAG_HAS_FLAGS) && (session.Flags&NBD_FLAG_SEND_FLUSH == NBD_FLAG_SEND_FLUSH)
}

func (session *NbdSession) Flush() error {
	if session.needFlush() {
		req := &NbdRequest{
			Magic:  NBD_REQUEST_MAGIC,
			Type:   NBD_CMD_FLUSH,
			Handle: session.Handle,
			From:   0,
			Len:    0,
		}
		_, err := session.Conn.Write(req.ToWireFormat())
		if err != nil {
			return err
		}
		return session.Recv()
	} else {
		return nil
	}
}

func (session *NbdSession) Disconnect() error {
	req := &NbdRequest{
		Magic:  NBD_REQUEST_MAGIC,
		Type:   NBD_CMD_DISC,
		Handle: session.Handle,
		From:   0,
		Len:    0,
	}
	_, err := session.Conn.Write(req.ToWireFormat())
	return err
}

func (session *NbdSession) Recv() error {
	_, err := session.Conn.Read(make([]byte, 4+4+8))
	session.Handle += 1
	return err
}
