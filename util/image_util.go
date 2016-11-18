package util

import (
	"fmt"
	"os"
	"os/exec"
)

func ConvertImageToQCOW2(imagePath string) error {
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return err
	}

	cmd := exec.Command("qemu-img", "convert", "-f", "raw", "-O", "qcow2", imagePath, imagePath+".qcow2")
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("Converting image %s to QCOW2 format failed in qemu-img\n", imagePath)
		return err
	}

	// Cleanup: remove raw image file first.
	if err := os.Remove(imagePath); err != nil {
		return err
	}

	// Finally, rename the QCOW2 file into the target appName.
	if err := os.Rename(imagePath+".qcow2", imagePath); err != nil {
		return err
	}

	return nil
}

func ResizeImage(imagePath string, targetSize uint64) error {
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return err
	}

	cmd := exec.Command("qemu-img", "resize", imagePath, fmt.Sprintf("%db", targetSize))
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("Resizing %s to new size %db failed in qemu-img\n", imagePath, targetSize)
		return err
	}

	return nil
}

func SetPartition(image string, partition int, start uint64, size uint64) error {
	partition = 0x1be + ((partition - 1) * 0x10)

	cyl, head, sec := chs(start / 512)
	cyl_end, head_end, sec_end := chs((start + size) / 512)

	nbdFile, err := NewNbdFile(image)
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
	nbdFile, err := NewNbdFile(imagePath)
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
