package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan"
	"github.com/cloudius-systems/capstan/image"
	"github.com/cloudius-systems/capstan/qemu"
	"os"
)

func Run(repo *capstan.Repo, verbose bool, imageName string) {
	var path string
	file, err := os.Open(imageName)
	if err == nil {
		path = imageName
		format := image.Probe(file)
		if format == image.Unknown {
			file.Close()
			fmt.Printf("%s: image format not recognized, unable to run it.\n", path)
			return
		}
		file.Close()
	} else {
		if !repo.ImageExists(imageName) {
			if !capstan.ConfigExists("Capstanfile") {
				fmt.Printf("%s: no such image\n", imageName)
				return
			}
			err := qemu.BuildImage(repo, imageName, verbose)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
		path = repo.ImagePath(imageName)
	}
	cmd := qemu.LaunchVM(true, path)
	cmd.Wait()
}
