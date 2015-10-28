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
