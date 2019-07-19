/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 * Modifications copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"github.com/cloudius-systems/capstan/util"
)

func Pull(r *util.Repo, hypervisor string, image string) error {
	remote, err := util.IsRemoteImage(r.URL, image)
	if err != nil {
		return err
	}
	if remote {
		return r.DownloadImage(hypervisor, image)
	}
	return r.PullImage(image)
}
