package main

import "github.com/penberg/capstan/repository"
import "github.com/penberg/capstan/qemu"
import "github.com/codegangsta/cli"
import "os"

var (
	VERSION string
)

func main() {
	app := cli.NewApp()
	app.Name = "capstan"
	app.Version = VERSION
	app.Usage = "pack, ship, and run applications in light-weight VMs"
	app.Commands = []cli.Command{
		{
			Name:      "push",
			Usage:     "push an image to a repository",
			Action: func(c *cli.Context) {
				repository.PushImage(c.Args().First())
			},
		},
		{
			Name:      "run",
			Usage:     "launch a VM",
			Action: func(c *cli.Context) {
				qemu.LaunchVM(c.Args().First())
			},
		},
		{
			Name:      "build",
			Usage:     "build an image",
			Action: func(c *cli.Context) {
				qemu.BuildImage(c.Args().First())
			},
		},
		{
			Name:      "images",
			Usage:     "list images",
			Action: func(c *cli.Context) {
				repository.ListImages()
			},
		},
	}
	app.Run(os.Args)
}
