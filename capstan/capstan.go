/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package main

import (
	"fmt"
	"github.com/cloudius-systems/capstan"
	"github.com/cloudius-systems/capstan/cmd"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/hypervisor"
	"github.com/cloudius-systems/capstan/instance"
	"github.com/codegangsta/cli"
	"os"
)

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
			Name:  "info",
			Usage: "show disk image information",
			Action: func(c *cli.Context) {
				if len(c.Args()) != 1 {
					fmt.Println("usage: capstan info [image-file]")
					return
				}
				image := c.Args()[0]
				cmd.Info(image)
			},
		},
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
			Usage: "pull an image from a repository",
			Flags: []cli.Flag{
				cli.StringFlag{"p", hypervisor.Default(), "hypervisor"},
			},
			Action: func(c *cli.Context) {
				if len(c.Args()) != 1 {
					fmt.Println("usage: capstan pull [image-name]")
					return
				}
				err := cmd.Pull(repo, c.String("p"), c.Args().First())
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:  "rmi",
			Usage: "delete an image from a repository",
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
			Flags: []cli.Flag{
				cli.StringFlag{"p", hypervisor.Default(), "hypervisor"},
				cli.StringFlag{"m", "1G", "memory size"},
				cli.IntFlag{"c", 2, "number of CPUs"},
				cli.BoolFlag{"v", "verbose mode"},
				cli.StringSliceFlag{"f", new(cli.StringSlice), "port forwarding rules"},
			},
			Action: func(c *cli.Context) {
				config := &cmd.RunConfig{
					ImageName:  c.Args().First(),
					Hypervisor: c.String("p"),
					Verbose:    c.Bool("v"),
					Memory:     c.String("m"),
					Cpus:       c.Int("c"),
					NatRules:   nat.Parse(c.StringSlice("f")),
				}
				err := cmd.Run(repo, config)
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:  "build",
			Usage: "build an image",
			Flags: []cli.Flag{
				cli.StringFlag{"p", hypervisor.Default(), "hypervisor"},
				cli.BoolFlag{"v", "verbose mode"},
			},
			Action: func(c *cli.Context) {
				image := c.Args().First()
				if len(c.Args()) != 1 {
					image = repo.DefaultImage()
				}
				if image == "" {
					fmt.Println("usage: capstan build [image-name]")
					return
				}
				hypervisor := c.String("p")
				err := cmd.Build(repo, hypervisor, image, c.Bool("v"))
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
		{
			Name:  "search",
			Usage: "search a remote images",
			Action: func(c *cli.Context) {
				image := ""
				if len(c.Args()) > 0 {
					image = c.Args()[0]
				}
				capstan.ListImagesRemote(image)
			},
		},
		{
			Name:  "instances",
			Usage: "list instances",
			Action: func(c *cli.Context) {
				instance.ListInstances()
			},
		},
	}
	app.Run(os.Args)
}
