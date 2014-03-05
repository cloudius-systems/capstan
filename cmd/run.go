package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan"
	"github.com/cloudius-systems/capstan/qemu"
)

func Run(repo *capstan.Repo, verbose bool, image string) {
	if !repo.ImageExists(image) {
		if !capstan.ConfigExists("Capstanfile") {
			fmt.Printf("%s: no such image\n", image)
			return
		}
		err := qemu.BuildImage(repo, image, verbose)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}
	file := repo.ImagePath(image)
	cmd := qemu.LaunchVM(true, file)
	cmd.Wait()
}
