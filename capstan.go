/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 * Modifications copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package main

import (
	"fmt"
	"os"

	"github.com/cloudius-systems/capstan/cmd"
	"github.com/cloudius-systems/capstan/core"
	"github.com/cloudius-systems/capstan/hypervisor"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/provider/openstack"
	"github.com/cloudius-systems/capstan/runtime"
	"github.com/cloudius-systems/capstan/util"
	"github.com/urfave/cli/v2"
)

var (
	VERSION string
)

const (
	// These exit codes were taken from BSD:
	// https://www.freebsd.org/cgi/man.cgi?query=sysexits&apropos=0&sektion=0&manpath=FreeBSD+4.3-RELEASE&format=html

	// The command was used incorrectly, e.g., with the wrong number of arguments,
	// a bad flag, a bad syntax in a parameter, or whatever.
	EX_USAGE = 64
	// The input data was incorrect in some way. This should only be used for
	// user's data & not system files.
	EX_DATAERR = 65
)

func main() {
	app := cli.NewApp()
	app.Name = "capstan"
	app.Version = VERSION
	app.Usage = "pack, ship, and run applications in light-weight VMs"
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "u", Usage: fmt.Sprintf("remote repository URL (default: \"%s\")", util.DefaultRepositoryUrl)},
		&cli.StringFlag{Name: "release-tag,r", Usage: "the release tag: any, latest, v0.51.0"},
		&cli.BoolFlag{Name: "s3", Usage: fmt.Sprintf("searches and downloads from S3 repository at (\"%s\")", util.DefaultRepositoryUrl)},
	}
	app.Commands = []*cli.Command{
		{
			Name:  "config",
			Usage: "Capstan configuration",
			Subcommands: []*cli.Command{
				{
					Name:  "print",
					Usage: "print current capstan configuration",
					Action: func(c *cli.Context) error {
						cmd.ConfigPrint(c)
						return nil
					},
				},
			},
		},
		{
			Name:  "info",
			Usage: "show disk image information",
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return cli.NewExitError("usage: capstan info [image-file]", EX_USAGE)
				}
				image := c.Args().Get(0)
				if err := cmd.Info(image); err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:  "import",
			Usage: "import an image to the local repository",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "v", Value: "", Usage: "image version"},
				&cli.StringFlag{Name: "c", Value: "", Usage: "image creation date"},
				&cli.StringFlag{Name: "d", Value: "", Usage: "image description"},
				&cli.StringFlag{Name: "b", Value: "", Usage: "image build command"},
			},
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 2 {
					return cli.NewExitError("usage: capstan import [image-name] [image-file]", EX_USAGE)
				}
				repo := util.NewRepoFromCli(c)
				err := repo.ImportImage(c.Args().Get(0), c.Args().Get(1), c.String("v"), c.String("c"), c.String("d"), c.String("b"))
				if err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:  "pull",
			Usage: "pull an image from a remote repository",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "p", Value: hypervisor.Default(), Usage: "hypervisor: qemu|vbox|vmw|gce"},
			},
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return cli.NewExitError("usage: capstan pull [image-name]", EX_USAGE)
				}
				hypervisor := c.String("p")
				if !isValidHypervisor(hypervisor) {
					return cli.NewExitError(fmt.Sprintf("error: '%s' is not a supported hypervisor\n", c.String("p")), EX_DATAERR)
				}
				repo := util.NewRepoFromCli(c)
				err := cmd.Pull(repo, hypervisor, c.Args().First())
				if err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:  "rmi",
			Usage: "delete an image from a repository",
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return cli.NewExitError("usage: capstan rmi [image-name]", EX_USAGE)
				}
				repo := util.NewRepoFromCli(c)
				err := repo.RemoveImage(c.Args().First())
				if err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:      "run",
			Usage:     "launch a VM. You may pass the image name as the first argument.",
			ArgsUsage: "instance-name",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "i", Value: "", Usage: "image_name"},
				&cli.StringFlag{Name: "p", Value: hypervisor.Default(), Usage: "hypervisor: qemu|vbox|vmw|gce|hkit"},
				&cli.StringFlag{Name: "m", Value: "1G", Usage: "memory size"},
				&cli.IntFlag{Name: "c", Value: 2, Usage: "number of CPUs"},
				&cli.StringFlag{Name: "n", Value: "nat", Usage: "networking: nat|bridge|tap|vhost|vnet|vpnkit"},
				&cli.BoolFlag{Name: "v", Usage: "verbose mode"},
				&cli.StringFlag{Name: "b", Value: "", Usage: "networking device (bridge or tap): e.g., virbr0, vboxnet0, tap0"},
				&cli.StringSliceFlag{Name: "f", Value: new(cli.StringSlice), Usage: "port forwarding rules"},
				&cli.StringFlag{Name: "gce-upload-dir", Value: "", Usage: "Directory to upload local image to: e.g., gs://osvimg"},
				&cli.StringFlag{Name: "mac", Value: "", Usage: "MAC address. If not specified, the MAC address will be generated automatically."},
				&cli.StringFlag{Name: "execute,e", Usage: "set the command line to execute"},
				&cli.StringSliceFlag{Name: "boot", Usage: "specify config_set name to boot unikernel with (repeatable, will be run left to right)"},
				&cli.BoolFlag{Name: "persist", Usage: "persist instance parameters (only relevant for qemu instances)"},
				&cli.StringSliceFlag{Name: "env", Value: new(cli.StringSlice), Usage: "specify value of environment variable e.g. PORT=8000 (repeatable)"},
				&cli.StringSliceFlag{Name: "volume", Value: new(cli.StringSlice), Usage: `{path}[:{key=val}], e.g. ./volume.img:format=raw (repeatable)
				Default options are :format=raw:aio=native:cache=none`},
			},
			Action: func(c *cli.Context) error {
				// Check for orphaned instances (those with osv.monitor and disk.qcow2, but
				// without osv.config) and remove them.
				if err := util.RemoveOrphanedInstances(c.Bool("v")); err != nil {
					return cli.NewExitError(err, EX_DATAERR)
				}

				bootOpts := cmd.BootOptions{
					Cmd:     c.String("execute"),
					Boot:    c.StringSlice("boot"),
					EnvList: c.StringSlice("env"),
				}
				bootCmd, err := bootOpts.GetCmd()
				if err != nil {
					return cli.NewExitError(err, EX_DATAERR)
				}

				config := &runtime.RunConfig{
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
					Cmd:          bootCmd,
					Persist:      c.Bool("persist"),
					Volumes:      c.StringSlice("volume"),
				}

				if !isValidHypervisor(config.Hypervisor) {
					return cli.NewExitError(fmt.Sprintf("error: '%s' is not a supported hypervisor\n", config.Hypervisor), EX_DATAERR)
				}
				repo := util.NewRepoFromCli(c)
				if err := cmd.RunInstance(repo, config); err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:  "build",
			Usage: "build an image",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "p", Value: hypervisor.Default(), Usage: "hypervisor: qemu|vbox|vmw|gce"},
				&cli.StringFlag{Name: "m", Value: "512M", Usage: "memory size"},
				&cli.BoolFlag{Name: "v", Usage: "verbose mode"},
			},
			Action: func(c *cli.Context) error {
				imageName := c.Args().First()
				repo := util.NewRepoFromCli(c)
				if c.Args().Len() != 1 {
					imageName = repo.DefaultImage()
				}
				if imageName == "" {
					return cli.NewExitError("usage: capstan build [image-name]", EX_USAGE)
				}
				hypervisor := c.String("p")
				if !isValidHypervisor(hypervisor) {
					return cli.NewExitError(fmt.Sprintf("error: '%s' is not a supported hypervisor\n", c.String("p")), EX_DATAERR)
				}
				image := &core.Image{
					Name:       imageName,
					Hypervisor: hypervisor,
				}
				template, err := core.ReadTemplateFile("Capstanfile")
				if err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				if err := cmd.Build(repo, image, template, c.Bool("v"), c.String("m")); err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:  "compose",
			Usage: "compose the image from a folder or a file",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "loader_image, l", Value: "osv-loader", Usage: "the base loader image"},
				&cli.StringFlag{Name: "size, s", Value: "10G", Usage: "size of the target user partition (use M or G suffix)"},
				&cli.StringFlag{Name: "command_line, c", Usage: "command line OSv will boot with"},
				&cli.BoolFlag{Name: "verbose, v", Usage: "verbose mode"},
			},
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 2 {
					return cli.NewExitError("Usage: capstan compose [image-name] [path-to-upload]", EX_USAGE)
				}

				// Name of the application (or image) that will be used in the internal repository.
				appName := c.Args().Get(0)
				// File or directory path that needs to be uploaded
				uploadPath := c.Args().Get(1)

				repo := util.NewRepoFromCli(c)

				loaderImage := c.String("l")

				commandLine := c.String("c")

				verbose := c.Bool("v")

				imageSize, err := util.ParseMemSize(c.String("size"))
				if err != nil {
					return cli.NewExitError(fmt.Sprintf("Incorrect image size format: %s\n", err), EX_DATAERR)
				}

				if err := cmd.Compose(repo, loaderImage, imageSize, uploadPath, appName, commandLine, verbose); err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:    "images",
			Aliases: []string{"i"},
			Usage:   "list images",
			Action: func(c *cli.Context) error {
				repo := util.NewRepoFromCli(c)
				fmt.Print(repo.ListImages())

				return nil
			},
		},
		{
			Name:  "search",
			Usage: "search a remote images",
			Action: func(c *cli.Context) error {
				image := ""
				if c.Args().Len() > 0 {
					image = c.Args().Get(0)
				}
				repo := util.NewRepoFromCli(c)
				err := util.ListImagesRemote(repo.URL, image)
				if err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:    "instances",
			Aliases: []string{"I"},
			Usage:   "list instances",
			Action: func(c *cli.Context) error {
				cmd.Instances()

				return nil
			},
		},
		{
			Name:  "stop",
			Usage: "stop an instance",
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return cli.NewExitError("usage: capstan stop [instance_name]", EX_USAGE)
				}
				instance := c.Args().Get(0)
				if err := cmd.Stop(instance); err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:  "delete",
			Usage: "delete an instance",
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return cli.NewExitError("usage: capstan delete [instance_name]", EX_USAGE)
				}
				instance := c.Args().Get(0)
				if err := cmd.Delete(instance); err != nil {
					return cli.NewExitError(err.Error(), EX_DATAERR)
				}
				return nil
			},
		},
		{
			Name:  "package",
			Usage: "package manipulation tools",
			Subcommands: []*cli.Command{
				{
					Name:      "init",
					Usage:     "initialise package structure",
					ArgsUsage: "[path]",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "name,n", Usage: "package name"},
						&cli.StringFlag{Name: "title,t", Usage: "package title"},
						&cli.StringFlag{Name: "author,a", Usage: "package author"},
						&cli.StringFlag{Name: "version,v", Usage: "package version"},
						&cli.StringSliceFlag{Name: "require", Usage: "specify package dependency"},
						&cli.StringFlag{Name: "runtime", Usage: "runtime to stub package for. Use 'capstan runtime list' to list all"},
						&cli.StringFlag{Name: "p, platform", Usage: "platform where package was built on"},
					},
					Action: func(c *cli.Context) error {
						if c.Args().Len() > 1 {
							return cli.NewExitError("usage: capstan package init [path]", EX_USAGE)
						}

						// The package path is the current working dir...
						packagePath, _ := os.Getwd()
						// ... unless the user has provided the exact location.
						if c.Args().Len() == 1 {
							packagePath = c.Args().Get(0)
						}

						// Author is a mandatory field.
						if c.String("name") == "" {
							return cli.NewExitError("You must provide the name of the package (--name or -n)", EX_USAGE)
						}

						// Author is a mandatory field.
						if c.String("title") == "" {
							return cli.NewExitError("You must provide the title of the package (--title or -t)", EX_USAGE)
						}

						// Author is a mandatory field.
						if c.String("author") == "" {
							return cli.NewExitError("You must provide the author of the package (--author or -a)", EX_USAGE)
						}

						// Initialise the package structure. The version may be empty as it is not
						// mandatory field.
						p := &core.Package{
							Name:     c.String("name"),
							Title:    c.String("title"),
							Author:   c.String("author"),
							Version:  c.String("version"),
							Require:  c.StringSlice("require"),
							Platform: c.String("platform"),
						}

						// Init package
						if err := cmd.InitPackage(packagePath, p); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						// Scaffold runtime if runtime name is provided
						if c.String("runtime") != "" {
							if err := cmd.RuntimeInit(c.String("runtime"), false, true); err != nil {
								return cli.NewExitError(err.Error(), EX_DATAERR)
							}
						}

						return nil
					},
				},
				{
					Name:  "build",
					Usage: "builds the package into a compressed file",
					Action: func(c *cli.Context) error {
						packageDir, _ := os.Getwd()

						_, err := cmd.BuildPackage(packageDir)
						if err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}
						return nil
					},
				},
				{
					Name:      "compose",
					Usage:     "composes the package and all its dependencies into OSv image",
					ArgsUsage: "image-name",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "size, s", Value: "10G", Usage: "total size of the target image (use M or G suffix)"},
						&cli.BoolFlag{Name: "update", Usage: "updates the existing target VM by uploading only modified files"},
						&cli.BoolFlag{Name: "verbose, v", Usage: "verbose mode"},
						&cli.StringFlag{Name: "run", Usage: "the command line to be executed in the VM"},
						&cli.BoolFlag{Name: "pull-missing, p", Usage: "attempt to pull packages missing from a local repository"},
						&cli.StringSliceFlag{Name: "boot", Usage: "specify default config_set name to boot unikernel with (repeatable, will be run left to right)"},
						&cli.StringSliceFlag{Name: "env", Value: new(cli.StringSlice), Usage: "specify value of environment variable e.g. PORT=8000 (repeatable)"},
						&cli.StringFlag{Name: "fs", Usage: "specify type of filesystem: zfs or rofs"},
						&cli.StringSliceFlag{Name: "require", Usage: "specify extra package dependency"},
						&cli.StringFlag{Name: "loader_image, l", Value: "osv-loader", Usage: "the base loader image"},
					},
					Action: func(c *cli.Context) error {
						if c.Args().Len() != 1 {
							return cli.NewExitError("Usage: capstan package compose [image-name]", EX_USAGE)
						}

						// Use the provided repository.
						repo := util.NewRepoFromCli(c)

						// Get the name of the application to be imported into Capstan's repository.
						appName := c.Args().First()

						// Parse image size descriptor.
						imageSize, err := util.ParseMemSize(c.String("size"))
						if err != nil {
							return cli.NewExitError(fmt.Sprintf("Incorrect image size format: %s\n", err), EX_USAGE)
						}

						loaderImage := c.String("l")

						updatePackage := c.Bool("update")
						verbose := c.Bool("verbose")
						pullMissing := c.Bool("pull-missing")

						// Always use the current directory for the package to compose.
						packageDir, _ := os.Getwd()

						bootOpts := cmd.BootOptions{
							Cmd:        c.String("run"),
							Boot:       c.StringSlice("boot"),
							EnvList:    c.StringSlice("env"),
							PackageDir: packageDir,
						}

						filesystem := c.String("fs")
						if filesystem == "" || (filesystem != "zfs" && filesystem != "rofs") {
							filesystem = "zfs"
						}

						if err := cmd.ComposePackage(repo, c.StringSlice("require"), imageSize, updatePackage, verbose, pullMissing,
							packageDir, appName, &bootOpts, filesystem, loaderImage); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:      "compose-remote",
					Usage:     "composes the package and all its dependencies and uploads resulting files into remote OSv instance",
					ArgsUsage: "remote-instance",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "verbose, v", Usage: "verbose mode"},
						&cli.BoolFlag{Name: "pull-missing, p", Usage: "attempt to pull packages missing from a local repository"},
						&cli.StringSliceFlag{Name: "require", Usage: "specify extra package dependency"},
					},
					Action: func(c *cli.Context) error {
						if c.Args().Len() != 1 {
							return cli.NewExitError("Usage: capstan package compose-remote [remote-instance]", EX_USAGE)
						}

						// Use the provided repository.
						repo := util.NewRepoFromCli(c)

						// Get the remote host instance or IP address where files of the composed package will be uploaded to
						remoteHostInstance := c.Args().First()

						verbose := c.Bool("verbose")
						pullMissing := c.Bool("pull-missing")

						// Always use the current directory for the package to compose.
						packageDir, _ := os.Getwd()

						if err := cmd.ComposePackageAndUploadToRemoteInstance(repo, c.StringSlice("require"), verbose, pullMissing,
							packageDir, remoteHostInstance); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:  "collect",
					Usage: "collects contents of this package and all required packages",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "pull-missing, p", Usage: "attempt to pull packages missing from a local repository"},
						&cli.BoolFlag{Name: "verbose, v", Usage: "verbose mode"},
						&cli.BoolFlag{Name: "remote", Usage: "set when previewing the compose-remote"},
						&cli.StringSliceFlag{Name: "require", Usage: "specify extra package dependency"},
					},
					Action: func(c *cli.Context) error {
						repo := util.NewRepoFromCli(c)
						packageDir, _ := os.Getwd()

						pullMissing := c.Bool("pull-missing")

						if err := cmd.CollectPackage(repo, packageDir, c.StringSlice("require"), pullMissing, c.Bool("remote"), c.Bool("verbose")); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:  "list",
					Usage: "lists the available packages",
					Action: func(c *cli.Context) error {
						repo := util.NewRepoFromCli(c)

						fmt.Print(repo.ListPackages())

						return nil
					},
				},
				{
					Name:  "import",
					Usage: "builds the package at the given path and imports it into a chosen repository",
					Action: func(c *cli.Context) error {
						// Use the provided repository.
						repo := util.NewRepoFromCli(c)

						packageDir, err := os.Getwd()
						if err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						if err = cmd.ImportPackage(repo, packageDir); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:      "search",
					Usage:     "searches for packages in the remote repository (partial name matches are also supported)",
					ArgsUsage: "[package-name]",
					Action: func(c *cli.Context) error {
						packageName := c.Args().First()

						repo := util.NewRepoFromCli(c)
						if err := repo.ListPackagesRemote(packageName); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:      "pull",
					Usage:     "pulls the package from remote repository and imports it into local package storage",
					ArgsUsage: "[package-name]",
					Action: func(c *cli.Context) error {
						// Name of the package is required argument.
						if c.Args().Len() != 1 {
							return cli.NewExitError("usage: capstan package pull [package-name]", EX_USAGE)
						}

						// Initialise the repository
						repo := util.NewRepoFromCli(c)
						if err := cmd.PullPackage(repo, c.Args().First()); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:      "describe",
					Usage:     "describes the package from local repository",
					ArgsUsage: "[package-name]",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "content, c", Usage: "show file content"},
					},
					Action: func(c *cli.Context) error {
						// Name of the package is required argument.
						if c.Args().Len() != 1 {
							return cli.NewExitError("usage: capstan package describe [package-name]", EX_USAGE)
						}

						// Initialise the repository
						repo := util.NewRepoFromCli(c)

						packageName := c.Args().Get(0)

						// Describe the package
						if s, err := cmd.DescribePackage(repo, packageName, c.Bool("content")); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						} else {
							fmt.Println(s)
						}

						return nil
					},
				},
				{
					Name:      "update",
					Usage:     "updates local packages from remote if remote version is newer",
					ArgsUsage: "[package-name]",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "created", Usage: "update package also if created date is newer (regardless version)"},
						&cli.BoolFlag{Name: "verbose, v", Usage: "verbose mode"},
					},
					Action: func(c *cli.Context) error {
						// Initialise the repository
						repo := util.NewRepoFromCli(c)

						search := ""
						if c.Args().Len() > 0 {
							search = c.Args().First()
						}

						// Update the package
						if err := cmd.UpdatePackages(repo, search, c.Bool("created"), c.Bool("verbose")); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
			},
		},
		{
			Name:    "stack",
			Aliases: []string{"openstack"},
			Usage:   "OpenStack manipulation tools",
			Subcommands: []*cli.Command{
				{
					Name:  "push",
					Usage: "composes OSv image and pushes it to OpenStack",
					Flags: append(
						[]cli.Flag{
							&cli.StringFlag{Name: "size, s", Value: "10G", Usage: "minimal size of the target user partition (use M or G suffix).\n" +
								"\tNOTE: will be enlarged to match flavor size."},
							&cli.StringFlag{Name: "flavor, f", Usage: "OpenStack flavor name that created OSv image should fit to"},
							&cli.StringFlag{Name: "run", Usage: "the command line to be executed in the VM"},
							&cli.BoolFlag{Name: "keep-image", Usage: "don't delete local composed image in .capstan/repository/stack"},
							&cli.BoolFlag{Name: "verbose, v", Usage: "verbose mode"},
							&cli.BoolFlag{Name: "pull-missing, p", Usage: "attempt to pull packages missing from a local repository"},
							&cli.StringSliceFlag{Name: "boot", Usage: "specify config_set name to boot unikernel with (repeatable, will be run left to right)"},
							&cli.StringSliceFlag{Name: "env", Value: new(cli.StringSlice), Usage: "specify value of environment variable e.g. PORT=8000 (repeatable)"},
						}, openstack.OPENSTACK_CREDENTIALS_FLAGS...),
					ArgsUsage:   "image-name",
					Description: "Compose package, build .qcow2 image and upload it to OpenStack under nickname <image-name>.",
					Action: func(c *cli.Context) error {
						err := cmd.OpenStackPush(c)
						if err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:  "run",
					Usage: "runs image that was previously pushed to OpenStack",
					Flags: append(
						[]cli.Flag{
							&cli.StringFlag{Name: "flavor, f", Usage: "OpenStack flavor to be run with"},
							&cli.StringFlag{Name: "mem, m", Usage: "MB of memory (RAM) to be run with"},
							&cli.StringFlag{Name: "name, n", Usage: "instance name"},
							&cli.IntFlag{Name: "count, c", Value: 1, Usage: "number of instances to run"},
							&cli.BoolFlag{Name: "verbose, v", Usage: "verbose mode"},
						}, openstack.OPENSTACK_CREDENTIALS_FLAGS...),
					ArgsUsage: "image-name",
					Description: "Run image that you've previously uploaded with 'capstan stack push'.\n   " +
						"Please note that image size CANNOT be changed at this point (wont' boot on\n   " +
						"too small flavor, wont use extra space on too big flavor), but feel free\n   " +
						"to adjust amount of memory (RAM).",
					Action: func(c *cli.Context) error {
						err := cmd.OpenStackRun(c)
						if err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
			},
		},
		{
			Name:  "runtime",
			Usage: "package runtime manipulation tools (meta/run.yaml)",
			Subcommands: []*cli.Command{
				{
					Name:  "preview",
					Usage: "prints runtime yaml template to the console",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "runtime, r", Usage: "Runtime name. Use 'capstan runtime list' to see available names."},
						&cli.BoolFlag{Name: "plain", Usage: "Remove comments"},
					},
					Action: func(c *cli.Context) error {
						if c.String("runtime") == "" {
							return cli.NewExitError("usage: capstan runtime preview -r [runtime-name]", EX_USAGE)
						}

						if err := cmd.RuntimePreview(c.String("runtime"), c.Bool("plain")); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:  "init",
					Usage: "prepares meta/run.yaml stub for selected runtime",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "runtime, r", Usage: "Runtime name. Use 'capstan runtime list' to see available names."},
						&cli.BoolFlag{Name: "plain", Usage: "Remove comments"},
						&cli.BoolFlag{Name: "force, f", Usage: "Override existing meta/run.yaml"},
					},
					Action: func(c *cli.Context) error {
						if c.String("runtime") == "" {
							return cli.NewExitError("usage: capstan runtime preview -r [runtime-name]", EX_USAGE)
						}

						if err := cmd.RuntimeInit(c.String("runtime"), c.Bool("plain"), c.Bool("force")); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:  "list",
					Usage: "list available runtimes",
					Flags: []cli.Flag{},
					Action: func(c *cli.Context) error {
						fmt.Print(cmd.RuntimeList())

						return nil
					},
				},
			},
		},
		{
			Name:  "volume",
			Usage: "volume manipulation tools",
			Subcommands: []*cli.Command{
				{
					Name:      "create",
					Usage:     "create empty volume and store it into ./volumes directory",
					ArgsUsage: "[volume-name]",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "size, s", Value: "1G", Usage: "total size of the target image (use M or G suffix)"},
						&cli.StringFlag{Name: "format, f", Usage: "volume format, e.g. [raw|qcow2|vdi|vmdk|...]"},
					},
					Action: func(c *cli.Context) error {
						if c.Args().Len() != 1 {
							return cli.NewExitError("usage: capstan volume create [volume-name]", EX_USAGE)
						}

						size, err := util.ParseMemSize(c.String("size"))
						if err != nil {
							return cli.NewExitError(fmt.Sprintf("Incorrect image size format: %s", err), EX_DATAERR)
						}

						volume := cmd.Volume{
							Volume: hypervisor.Volume{
								Format: c.String("format"),
							},
							SizeMB: size,
							Name:   c.Args().First(),
						}
						packagePath, _ := os.Getwd()
						if err := cmd.CreateVolume(packagePath, volume); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
				{
					Name:      "delete",
					Usage:     "delete volume by name",
					ArgsUsage: "[volume-name]",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "verbose, v", Usage: "verbose mode"},
					},
					Action: func(c *cli.Context) error {
						if c.Args().Len() != 1 {
							return cli.NewExitError("usage: capstan volume delete [volume-name]", EX_USAGE)
						}
						name := c.Args().First()

						packagePath, _ := os.Getwd()
						if err := cmd.DeleteVolume(packagePath, name, c.Bool("verbose")); err != nil {
							return cli.NewExitError(err.Error(), EX_DATAERR)
						}

						return nil
					},
				},
			},
		},
	}
	app.Run(os.Args)
}

func isValidHypervisor(hypervisor string) bool {
	switch hypervisor {
	case "qemu", "vbox", "vmw", "gce", "hkit":
		return true
	default:
		return false
	}
}
