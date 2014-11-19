/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core

import (
	"fmt"
	"os"
	"os/exec"
)

type RpmPackage struct {
	Name    string
	Version string
	Release string
	Arch    string
}

func (p *RpmPackage) Download() error {
	if _, err := os.Stat(p.Filename()); os.IsNotExist(err) {
		fmt.Printf("Downloading %s...\n", p.Filename())
		cmd := exec.Command("curl", "-O", p.URL())
		_, err = cmd.Output()
		if err != nil {
			return err
		}

	}
	return nil
}

func (p *RpmPackage) URL() string {
	baseUrl := "http://kojipkgs.fedoraproject.org/packages/"
	return fmt.Sprintf("%s%s/%s/%s/%s/%s", baseUrl, p.Name, p.Version, p.Release, p.Arch, p.Filename())
}

func (p *RpmPackage) Filename() string {
	return fmt.Sprintf("%s-%s-%s.%s.rpm", p.Name, p.Version, p.Release, p.Arch)
}
