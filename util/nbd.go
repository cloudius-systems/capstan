/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 * Modifications copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
)

const (
	NBD_REQUEST_MAGIC = 0x25609513
	NBD_REPLY_MAGIC   = 0x67446698
	NBD_OLD_STYLE_HANDSHAKE_MAGIC = 0x00420281861253
	NBD_NEW_STYLE_HANDSHAKE_MAGIC = 0x49484156454F5054
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

type NbdFile struct {
	Cmd     *exec.Cmd
	Session *NbdSession
}

type NbdSession struct {
	Conn   net.Conn
	Handle uint64
	Size   uint64
	Flags  uint32
	Req    *NbdRequest
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

func NewNbdFile(imagePath string) (*NbdFile, error) {
	cmd := exec.Command("qemu-nbd", "-p", "10809", imagePath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	conn, err := ConnectAndWait("tcp", "localhost:10809")
	if err != nil {
		return nil, err
	}

	session := &NbdSession{
		Conn:   conn,
		Handle: 0,
	}

	if err := session.Handshake(); err != nil {
		return nil, err
	}

	return &NbdFile{cmd, session}, nil
}

func (file *NbdFile) Write(offset uint64, data []byte) error {
	count := uint64(len(data))
	sectStart := (offset / 512) * 512
	offsetInSect := offset % 512

	size := offsetInSect + count
	sectSize := ((size / 512) + 1) * 512

	readData, err := file.Session.Read(sectStart, uint32(sectSize))
	if err != nil {
		return err
	}

	buf := append(append(readData[0:offsetInSect], data...), readData[offsetInSect+count:]...)
	err = file.Session.Write(sectStart, buf)
	if err != nil {
		return err
	}

	return file.Session.Flush()
}

func (file *NbdFile) WriteByte(offset uint64, b byte) error {
	buf := bytes.Buffer{}

	err := binary.Write(&buf, binary.LittleEndian, b)
	if err != nil {
		return err
	}

	return file.Write(offset, buf.Bytes())
}

func (file *NbdFile) WriteShort(offset uint64, s uint16) error {
	buf := bytes.Buffer{}

	err := binary.Write(&buf, binary.LittleEndian, s)
	if err != nil {
		return err
	}

	return file.Write(offset, buf.Bytes())
}

func (file *NbdFile) WriteInt(offset uint64, i uint32) error {
	buf := bytes.Buffer{}

	err := binary.Write(&buf, binary.LittleEndian, i)
	if err != nil {
		return err
	}

	return file.Write(offset, buf.Bytes())
}

func (file *NbdFile) Wait() {
	file.Cmd.Wait()
}

func (file *NbdFile) Close() error {
	if err := file.Session.Flush(); err != nil {
		return err
	}
	if err := file.Session.Disconnect(); err != nil {
		return err
	}
	file.Session.Conn.Close()
	file.Wait()

	return nil
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
	buf := make([]byte, 8)
	session.Conn.Read(buf)
	magic := binary.BigEndian.Uint64(buf)
	if magic == NBD_NEW_STYLE_HANDSHAKE_MAGIC {
		return session.NewStyleHandshake(magic)
	} else {
		return session.OldStyleHandshake(magic)
	}
}

func (session *NbdSession) OldStyleHandshake(magic uint64) error {
	if magic != NBD_OLD_STYLE_HANDSHAKE_MAGIC {
		return fmt.Errorf("Bad magic: %x! Expected %x instead!", magic, NBD_OLD_STYLE_HANDSHAKE_MAGIC)
	}
	buf := make([]byte, 8+4)
	session.Conn.Read(buf)
	session.Size = binary.BigEndian.Uint64(buf)
	session.Flags = binary.BigEndian.Uint32(buf)
	session.Conn.Read(make([]byte, 124))
	session.Handle += 1
	return nil
}

func (session *NbdSession) NewStyleHandshake(magic uint64) error {
	if magic != NBD_NEW_STYLE_HANDSHAKE_MAGIC {
		return fmt.Errorf("Bad magic: %x! Expected %x instead!", magic, NBD_NEW_STYLE_HANDSHAKE_MAGIC)
	}
	buf := make([]byte, 2)
	session.Conn.Read(buf)
	handshakeFlags := binary.BigEndian.Uint16(buf)

	endian := binary.BigEndian
	b := make([]byte, 4+8+4+4)
	endian.PutUint32(b[0:4], 0)
	endian.PutUint64(b[4:12], NBD_NEW_STYLE_HANDSHAKE_MAGIC)
	endian.PutUint32(b[12:16], 1) //NBD_OPT_EXPORT_NAME
	endian.PutUint32(b[16:20], 0) //No option data
	session.Conn.Write(b)

	buf = make([]byte, 8+2)
	session.Conn.Read(buf)
	session.Size = binary.BigEndian.Uint64(buf)
	transportFlags := binary.BigEndian.Uint16(buf)
	session.Flags = uint32(handshakeFlags) << 16 + uint32(transportFlags)

	session.Conn.Read(make([]byte, 124))
	session.Handle += 1
	return nil
}

func (session *NbdSession) Write(from uint64, data []byte) error {
	req := &NbdRequest{
		Magic:  NBD_REQUEST_MAGIC,
		Type:   NBD_CMD_WRITE,
		Handle: session.Handle,
		From:   from,
		Len:    uint32(len(data)),
	}

	session.Req = req

	_, err := session.Conn.Write(append(req.ToWireFormat(), data...))
	if err != nil {
		return err
	}

	_, err = session.Recv()
	return err
}

func (session *NbdSession) Read(offset uint64, length uint32) ([]byte, error) {
	req := &NbdRequest{
		Magic:  NBD_REQUEST_MAGIC,
		Type:   NBD_CMD_READ,
		Handle: session.Handle,
		From:   offset,
		Len:    length,
	}

	session.Req = req

	_, err := session.Conn.Write(req.ToWireFormat())
	if err != nil {
		return nil, err
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

		_, err = session.Recv()

		return err
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

func (session *NbdSession) Recv() ([]byte, error) {
	_, err := session.Conn.Read(make([]byte, 4+4+8))
	session.Handle += 1

	if session.Req.Type == NBD_CMD_READ {
		data := make([]byte, session.Req.Len)
		_, err := session.Conn.Read(data)
		if err != nil {
			return nil, err
		}

		return data, nil
	}

	return nil, err
}
