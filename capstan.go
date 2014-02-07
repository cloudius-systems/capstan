/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package main

import "github.com/cloudius-systems/capstan/repository"
import "github.com/cloudius-systems/capstan/qemu"
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
			Name:  "push",
			Usage: "push an image to a repository",
			Action: func(c *cli.Context) {
				repository.PushImage(c.Args().First())
			},
		},
		{
			Name:  "rmi",
			Usage: "delete an image from an repository",
			Action: func(c *cli.Context) {
				repository.RemoveImage(c.Args().First())
			},
		},
		{
			Name:  "run",
			Usage: "launch a VM",
			Action: func(c *cli.Context) {
				cmd := qemu.LaunchVM(c.Args().First())
				cmd.Wait()
			},
		},
		{
			Name:  "build",
			Usage: "build an image",
			Action: func(c *cli.Context) {
				qemu.BuildImage(c.Args().First())
			},
		},
		{
			Name:  "images",
			Usage: "list images",
			Action: func(c *cli.Context) {
				repository.ListImages()
			},
		},
	}
	app.Run(os.Args)
}
