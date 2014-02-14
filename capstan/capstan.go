/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package main

import "fmt"
import "github.com/cloudius-systems/capstan"
import "github.com/cloudius-systems/capstan/qemu"
import "github.com/codegangsta/cli"
import "os"

var (
	VERSION string
)

func main() {
	repo := capstan.NewRepo()
	app := cli.NewApp()
	app.Name = "capstan"
	app.Version = VERSION
	app.Usage = "pack, ship, and run applications in light-weight VMs"
	app.Commands = []cli.Command{
		{
			Name:  "push",
			Usage: "push an image to a repository",
			Action: func(c *cli.Context) {
				if len(c.Args()) != 2 {
					fmt.Println("usage: capstan push [image-name] [image-file]")
					return
				}
				err := repo.PushImage(c.Args()[0], c.Args()[1])
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:  "pull",
			Usage: "pull an image to the repository",
			Action: func(c *cli.Context) {
				if len(c.Args()) != 1 {
					fmt.Println("usage: capstan pull [image-name]")
					return
				}
				err := repo.PullImage(c.Args().First())
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:  "rmi",
			Usage: "delete an image from an repository",
			Action: func(c *cli.Context) {
				if len(c.Args()) != 1 {
					fmt.Println("usage: capstan rmi [image-name]")
					return
				}
				err := repo.RemoveImage(c.Args().First())
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:  "run",
			Usage: "launch a VM",
			Action: func(c *cli.Context) {
				image := c.Args().First()
				if len(c.Args()) != 1 {
					image = repo.DefaultImage()
				}
				if image == "" {
					fmt.Println("usage: capstan run [image-name]")
					return
				}
				if !repo.ImageExists(image) {
					err := qemu.BuildImage(repo, image)
					if err != nil {
						fmt.Println(err.Error())
						return
					}
				}
				cmd := qemu.LaunchVM(repo, image)
				cmd.Wait()
			},
		},
		{
			Name:  "build",
			Usage: "build an image",
			Action: func(c *cli.Context) {
				image := c.Args().First()
				if len(c.Args()) != 1 {
					image = repo.DefaultImage()
				}
				if image == "" {
					fmt.Println("usage: capstan build [image-name]")
					return
				}
				err := qemu.BuildImage(repo, image)
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:  "images",
			Usage: "list images",
			Action: func(c *cli.Context) {
				repo.ListImages()
			},
		},
	}
	app.Run(os.Args)
}
