/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package main

import (
	"fmt"
	"github.com/cloudius-systems/capstan/cmd"
	"github.com/cloudius-systems/capstan/core"
	"github.com/cloudius-systems/capstan/hypervisor"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/util"
	"github.com/codegangsta/cli"
	"os"
)

var (
	VERSION string
)

const (
	DEFAULT_REPO_URL = "https://s3.amazonaws.com/osv.capstan/"
)

func main() {
	app := cli.NewApp()
	app.Name = "capstan"
	app.Version = VERSION
	app.Usage = "pack, ship, and run applications in light-weight VMs"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "u", Value: DEFAULT_REPO_URL, Usage: "remote repository URL"},
	}
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
				err := cmd.Info(image)
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:  "import",
			Usage: "import an image to the local repository",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "v", Value: "", Usage: "image version"},
				cli.StringFlag{Name: "c", Value: "", Usage: "image creation date"},
				cli.StringFlag{Name: "d", Value: "", Usage: "image description"},
				cli.StringFlag{Name: "b", Value: "", Usage: "image build command"},
			},
			Action: func(c *cli.Context) {
				if len(c.Args()) != 2 {
					fmt.Println("usage: capstan import [image-name] [image-file]")
					return
				}
				repo := util.NewRepo(c.GlobalString("u"))
				err := repo.ImportImage(c.Args()[0], c.Args()[1], c.String("v"), c.String("c"), c.String("d"), c.String("b"))
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:  "pull",
			Usage: "pull an image from a repository",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "p", Value: hypervisor.Default(), Usage: "hypervisor: qemu|vbox|vmw|gce"},
			},
			Action: func(c *cli.Context) {
				if len(c.Args()) != 1 {
					fmt.Println("usage: capstan pull [image-name]")
					return
				}
				hypervisor := c.String("p")
				if !isValidHypervisor(hypervisor) {
					fmt.Printf("error: '%s' is not a supported hypervisor\n", c.String("p"))
					return
				}
				repo := util.NewRepo(c.GlobalString("u"))
				err := cmd.Pull(repo, hypervisor, c.Args().First())
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
				repo := util.NewRepo(c.GlobalString("u"))
				err := repo.RemoveImage(c.Args().First())
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:      "run",
			Usage:     "launch a VM. You may pass the image name as the first argument.",
			ArgsUsage: "instance-name",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "i", Value: "", Usage: "image_name"},
				cli.StringFlag{Name: "p", Value: hypervisor.Default(), Usage: "hypervisor: qemu|vbox|vmw|gce"},
				cli.StringFlag{Name: "m", Value: "1G", Usage: "memory size"},
				cli.IntFlag{Name: "c", Value: 2, Usage: "number of CPUs"},
				cli.StringFlag{Name: "n", Value: "nat", Usage: "networking: nat|bridge|tap"},
				cli.BoolFlag{Name: "v", Usage: "verbose mode"},
				cli.StringFlag{Name: "b", Value: "", Usage: "networking device (bridge or tap): e.g., virbr0, vboxnet0, tap0"},
				cli.StringSliceFlag{Name: "f", Value: new(cli.StringSlice), Usage: "port forwarding rules"},
				cli.StringFlag{Name: "gce-upload-dir", Value: "", Usage: "Directory to upload local image to: e.g., gs://osvimg"},
				cli.StringFlag{Name: "mac", Value: "", Usage: "MAC address. If not specified, the MAC address will be generated automatically."},
				cli.StringFlag{Name: "execute,e", Usage: "set the command line to execute"},
			},
			Action: func(c *cli.Context) {
				config := &cmd.RunConfig{
					InstanceName: c.Args().First(),
					ImageName:    c.String("i"),
					Hypervisor:   c.String("p"),
					Verbose:      c.Bool("v"),
					Memory:       c.String("m"),
					Cpus:         c.Int("c"),
					Networking:   c.String("n"),
					Bridge:       c.String("b"),
					NatRules:     nat.Parse(c.StringSlice("f")),
					GCEUploadDir: c.String("gce-upload-dir"),
					MAC:          c.String("mac"),
					Cmd:          c.String("execute"),
				}
				if !isValidHypervisor(config.Hypervisor) {
					fmt.Printf("error: '%s' is not a supported hypervisor\n", config.Hypervisor)
					return
				}
				repo := util.NewRepo(c.GlobalString("u"))
				err := cmd.RunInstance(repo, config)
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:  "build",
			Usage: "build an image",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "p", Value: hypervisor.Default(), Usage: "hypervisor: qemu|vbox|vmw|gce"},
				cli.StringFlag{Name: "m", Value: "512M", Usage: "memory size"},
				cli.BoolFlag{Name: "v", Usage: "verbose mode"},
			},
			Action: func(c *cli.Context) {
				imageName := c.Args().First()
				repo := util.NewRepo(c.GlobalString("u"))
				if len(c.Args()) != 1 {
					imageName = repo.DefaultImage()
				}
				if imageName == "" {
					fmt.Println("usage: capstan build [image-name]")
					return
				}
				hypervisor := c.String("p")
				if !isValidHypervisor(hypervisor) {
					fmt.Printf("error: '%s' is not a supported hypervisor\n", c.String("p"))
					return
				}
				image := &core.Image{
					Name:       imageName,
					Hypervisor: hypervisor,
				}
				template, err := core.ReadTemplateFile("Capstanfile")
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				if err := cmd.Build(repo, image, template, c.Bool("v"), c.String("m")); err != nil {
					fmt.Println(err.Error())
					return
				}
			},
		},
		{
			Name:  "compose",
			Usage: "compose the image from a folder or a file",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "loader_image, l", Value: "mike/osv-loader", Usage: "the base loader image"},
				cli.StringFlag{Name: "size, s", Value: "10G", Usage: "size of the target user partition (use M or G suffix)"},
			},
			Action: func(c *cli.Context) {
				if len(c.Args()) != 2 {
					fmt.Println("Usage: capstan compose [image-name] [path-to-upload]")
					return
				}

				// Name of the application (or image) that will be used in the internal repository.
				appName := c.Args()[0]
				// File or directory path that needs to be uploaded
				uploadPath := c.Args()[1]

				repo := util.NewRepo(c.GlobalString("u"))

				loaderImage := c.String("l")

				imageSize, err := util.ParseMemSize(c.String("size"))
				if err != nil {
					fmt.Printf("Incorrect image size format: %s\n", err)
					return
				}

				if err := cmd.Compose(repo, loaderImage, imageSize, uploadPath, appName); err != nil {
					fmt.Println(err.Error())
					return
				}
			},
		},
		{
			Name:      "images",
			ShortName: "i",
			Usage:     "list images",
			Action: func(c *cli.Context) {
				repo := util.NewRepo(c.GlobalString("u"))
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
				err := util.ListImagesRemote(c.GlobalString("u"), image)
				if err != nil {
					fmt.Println(err.Error())
				}
			},
		},
		{
			Name:      "instances",
			ShortName: "I",
			Usage:     "list instances",
			Action: func(c *cli.Context) {
				cmd.Instances()
			},
		},
		{
			Name:  "stop",
			Usage: "stop an instance",
			Action: func(c *cli.Context) {
				if len(c.Args()) != 1 {
					fmt.Println("usage: capstan stop [instance_name]")
					return
				}
				instance := c.Args()[0]
				cmd.Stop(instance)
			},
		},
		{
			Name:  "delete",
			Usage: "delete an instance",
			Action: func(c *cli.Context) {
				if len(c.Args()) != 1 {
					fmt.Println("usage: capstan delete [instance_name]")
					return
				}
				instance := c.Args()[0]
				cmd.Delete(instance)
			},
		},
		{
			Name:  "package",
			Usage: "package manipulation tools",
			Subcommands: []cli.Command{
				{
					Name:      "init",
					Usage:     "initialise package structure",
					ArgsUsage: "[path]",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "name,n", Usage: "package name"},
						cli.StringFlag{Name: "title,t", Usage: "package title"},
						cli.StringFlag{Name: "author,a", Usage: "package author"},
						cli.StringFlag{Name: "version,v", Usage: "package version"},
						cli.StringSliceFlag{Name: "require", Usage: "specify package dependency"},
					},
					Action: func(c *cli.Context) {
						if len(c.Args()) > 1 {
							fmt.Println("usage: capstan package init [path]")
							return
						}

						// The package path is the current working dir...
						packagePath, _ := os.Getwd()
						// ... unless the user has provided the exact location.
						if len(c.Args()) == 1 {
							packagePath = c.Args()[0]
						}

						// Author is a mandatory field.
						if c.String("name") == "" {
							fmt.Println("You must provide the name of the package (--name or -n)")
							return
						}

						// Author is a mandatory field.
						if c.String("title") == "" {
							fmt.Println("You must provide the title of the package (--title or -t)")
							return
						}

						// Author is a mandatory field.
						if c.String("author") == "" {
							fmt.Println("You must provide the author of the package (--author or -a)")
							return
						}

						// Initialise the package structure. The version may be empty as it is not
						// mandatory field.
						p := &core.Package{
							Name:    c.String("name"),
							Title:   c.String("title"),
							Author:  c.String("author"),
							Version: c.String("version"),
							Require: c.StringSlice("require"),
						}

						cmd.InitPackage(packagePath, p)
					},
				},
				{
					Name:  "build",
					Usage: "builds the package into a compressed file",
					Action: func(c *cli.Context) {
						packageDir, _ := os.Getwd()

						_, err := cmd.BuildPackage(packageDir)
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
					},
				},
				{
					Name:      "compose",
					Usage:     "composes the package and all its dependencies into OSv image",
					ArgsUsage: "image-name",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "size, s", Value: "10G", Usage: "size of the target user partition (use M or G suffix)"},
						cli.BoolFlag{Name: "update", Usage: "updates the existing target VM by uploading only modified files"},
						cli.BoolFlag{Name: "verbose, v", Usage: "verbose mode"},
					},
					Action: func(c *cli.Context) {
						if len(c.Args()) != 1 {
							fmt.Println("Usage: capstan package compose [image-name]")
							return
						}

						// Use the provided repository.
						repo := util.NewRepo(c.GlobalString("u"))

						// Get the name of the application to be imported into Capstan's repository.
						appName := c.Args().First()

						// Parse image size descriptor.
						imageSize, err := util.ParseMemSize(c.String("size"))
						if err != nil {
							fmt.Printf("Incorrect image size format: %s\n", err)
							return
						}

						updatePackage := c.Bool("update")
						verbose := c.Bool("verbose")

						// Always use the current directory for the package to compose.
						packageDir, _ := os.Getwd()

						if err := cmd.ComposePackage(repo, imageSize, updatePackage, verbose, packageDir, appName); err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
					},
				},
				{
					Name:  "collect",
					Usage: "collects contents of this package and all required packages",
					Action: func(c *cli.Context) {
						repo := util.NewRepo(c.GlobalString("u"))
						packageDir, _ := os.Getwd()

						if err := cmd.CollectPackage(repo, packageDir); err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
					},
				},
				{
					Name:  "list",
					Usage: "lists the available packages",
					Action: func(c *cli.Context) {
						repo := util.NewRepo(c.GlobalString("u"))

						repo.ListPackages()
					},
				},
				{
					Name:  "import",
					Usage: "builds the package at the given path and imports it into a chosen repository",
					Action: func(c *cli.Context) {
						// Use the provided repository.
						repo := util.NewRepo(c.GlobalString("u"))

						packageDir, err := os.Getwd()
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}

						if err = cmd.ImportPackage(repo, packageDir); err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
					},
				},
			},
		},
	}
	app.Run(os.Args)
}

func isValidHypervisor(hypervisor string) bool {
	switch hypervisor {
	case "qemu", "vbox", "vmw", "gce":
		return true
	default:
		return false
	}
}
