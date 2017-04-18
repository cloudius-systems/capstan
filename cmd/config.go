/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"fmt"

	"github.com/mikelangelo-project/capstan/core"
	"github.com/mikelangelo-project/capstan/util"
	"github.com/urfave/cli"
)

// ConfigPrint prints current capstan configuration to console.
func ConfigPrint(c *cli.Context) error {
	repo := util.NewRepo(c.GlobalString("u"))
	fmt.Println("--- global configuration")
	repo.PrintRepo()
	fmt.Println()

	fmt.Println("--- curent directory configuration")
	fmt.Println("CAPSTANIGNORE:")
	capstanignore := core.CapstanignoreInit("./.capstanignore")
	capstanignore.PrintPatterns()

	return nil
}
