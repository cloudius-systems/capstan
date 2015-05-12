/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"github.com/cloudius-systems/capstan/util"
)

func Pull(r *util.Repo, hypervisor string, image string) error {
	if util.IsRemoteImage(r.URL, image) {
		return r.DownloadImage(r.URL, hypervisor, image)
	}
	return r.PullImage(image)
}
