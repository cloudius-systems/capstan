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

type NbdSession struct {
	Conn   net.Conn
	Handle uint64
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
	session.Conn.Read(make([]byte, 8+8+4))
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

func (session *NbdSession) Flush() error {
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
