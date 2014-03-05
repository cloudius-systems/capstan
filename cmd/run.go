package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan"
	"github.com/cloudius-systems/capstan/image"
	"github.com/cloudius-systems/capstan/qemu"
	"os"
)

type RunConfig struct {
	ImageName string
	Verbose   bool
}

func Run(repo *capstan.Repo, config *RunConfig) {
	var path string
	file, err := os.Open(config.ImageName)
	if err == nil {
		path = config.ImageName
		format := image.Probe(file)
		if format == image.Unknown {
			file.Close()
			fmt.Printf("%s: image format not recognized, unable to run it.\n", path)
			return
		}
		file.Close()
	} else {
		if !repo.ImageExists(config.ImageName) {
			if !capstan.ConfigExists("Capstanfile") {
				fmt.Printf("%s: no such image\n", config.ImageName)
				return
			}
			err := qemu.BuildImage(repo, config.ImageName, config.Verbose)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
		path = repo.ImagePath(config.ImageName)
	}
	cmd := qemu.LaunchVM(true, path)
	cmd.Wait()
}
