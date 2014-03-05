package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan/image"
	"os"
)

func Info(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	format := image.Probe(f)
	switch format {
	case image.VDI:
		fmt.Printf("%s: VDI\n", path)
	case image.QCOW2:
		fmt.Printf("%s: QCOW2\n", path)
	default:
		fmt.Printf("%s: not a runnable image\n", path)
	}
	return nil
}
