/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"github.com/cloudius-systems/capstan"
)

func Pull(r *capstan.Repo, image string) error {
	if capstan.IsRemoteImage(image) {
		return r.DownloadImage(image)
	}
	return r.PullImage(image)
}
