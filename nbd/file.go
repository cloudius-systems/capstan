package nbd

import (
	"bytes"
	"encoding/binary"
	"github.com/cloudius-systems/capstan/util"
	"io"
	"os"
	"os/exec"
)

type File struct {
	Cmd     *exec.Cmd
	Session *NbdSession
}

func NewFile(imagePath string) (*File, error) {
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

	conn, err := util.ConnectAndWait("tcp", "localhost:10809")
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

	return &File{cmd, session}, nil
}

func (file *File) Write(offset uint64, data []byte) error {
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

func (file *File) WriteByte(offset uint64, b byte) error {
	buf := bytes.Buffer{}

	err := binary.Write(&buf, binary.LittleEndian, b)
	if err != nil {
		return err
	}

	return file.Write(offset, buf.Bytes())
}

func (file *File) WriteShort(offset uint64, s uint16) error {
	buf := bytes.Buffer{}

	err := binary.Write(&buf, binary.LittleEndian, s)
	if err != nil {
		return err
	}

	return file.Write(offset, buf.Bytes())
}

func (file *File) WriteInt(offset uint64, i uint32) error {
	buf := bytes.Buffer{}

	err := binary.Write(&buf, binary.LittleEndian, i)
	if err != nil {
		return err
	}

	return file.Write(offset, buf.Bytes())
}

func (file *File) Wait() {
	file.Cmd.Wait()
}

func (file *File) Close() error {
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

func SetPartition(image string, partition int, start uint64, size uint64) error {
	partition = 0x1be + ((partition - 1) * 0x10)

	cyl, head, sec := chs(start / 512)
	cyl_end, head_end, sec_end := chs((start + size) / 512)

	nbdFile, err := NewFile(image)
	if err != nil {
		return err
	}

	if err := nbdFile.WriteByte(uint64(partition+1), byte(head)); err != nil {
		return err
	}
	if err := nbdFile.WriteByte(uint64(partition+5), byte(head_end)); err != nil {
		return err
	}
	if err := nbdFile.WriteShort(uint64(partition+2), uint16(cyl<<6|sec)); err != nil {
		return err
	}
	if err := nbdFile.WriteShort(uint64(partition+6), uint16(cyl_end<<6|sec_end)); err != nil {
		return err
	}

	systemId := 0x83
	if err := nbdFile.WriteByte(uint64(partition+4), byte(systemId)); err != nil {
		return err
	}

	if err := nbdFile.WriteInt(uint64(partition+8), uint32(start/512)); err != nil {
		return err
	}
	if err := nbdFile.WriteInt(uint64(partition+12), uint32(size/512)); err != nil {
		return err
	}

	if err := nbdFile.Close(); err != nil {
		return err
	}

	return nil
}

func SetCmdLine(imagePath string, cmdLine string) error {
	nbdFile, err := NewFile(imagePath)
	if err != nil {
		return err
	}

	padding := 512 - (len(cmdLine) % 512)

	data := append([]byte(cmdLine), make([]byte, padding)...)

	if err := nbdFile.Write(512, data); err != nil {
		return err
	}

	if err := nbdFile.Close(); err != nil {
		return err
	}

	return nil
}

func chs(x uint64) (uint64, uint64, uint64) {
	sectorsPerTrack := uint64(63)
	heads := uint64(255)

	c := (x / sectorsPerTrack) / heads
	h := (x / sectorsPerTrack) % heads
	s := (x % sectorsPerTrack) + 1

	if c > 1023 {
		c = 1023
		h = 254
		s = 63
	}

	return c, h, s
}
