/*
 * Copyright (C) 2018 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"github.com/cloudius-systems/capstan/core"
	"time"

	. "gopkg.in/check.v1"
)

type s3repoSuite struct{}

var _ = Suite(&s3repoSuite{})

func (*s3repoSuite) TestNeedsUpdate(c *C) {
	m := []struct {
		comment             string
		localPkg            core.Package
		localCreated        string
		remotePkg           core.Package
		remoteCreated       string
		compareCreated      bool
		expectedNeedsUpdate bool
	}{
		{
			"regular case",
			core.Package{Version: "4.12.10"}, "",
			core.Package{Version: "5.0.0"}, "",
			false,
			true,
		},
		{
			"already latest",
			core.Package{Version: "4.12.10"}, "",
			core.Package{Version: "4.12.10"}, "",
			false,
			false,
		},
		{
			"local ahead of remote",
			core.Package{Version: "4.12.10"}, "",
			core.Package{Version: "3.0.0"}, "",
			false,
			false,
		},
		{
			"needs update #major",
			core.Package{Version: "4.12.10"}, "",
			core.Package{Version: "5.12.10"}, "",
			false,
			true,
		},
		{
			"needs update #minor",
			core.Package{Version: "4.12.10"}, "",
			core.Package{Version: "4.13.10"}, "",
			false,
			true,
		},
		{
			"needs update #patch",
			core.Package{Version: "4.12.10"}, "",
			core.Package{Version: "4.13.11"}, "",
			false,
			true,
		},
		{
			"update because of time created",
			core.Package{Version: "4.12.10"}, "2018-01-05 07:44",
			core.Package{Version: "4.12.10"}, "2018-01-05 07:45",
			true,
			true,
		},
		{
			"both version and time created are latest",
			core.Package{Version: "4.12.10"}, "2018-01-05 07:44",
			core.Package{Version: "4.12.10"}, "2018-01-05 07:44",
			true,
			false,
		},
		{
			"time created invalid",
			core.Package{Version: "4.12.10"}, "invalid",
			core.Package{Version: "4.12.10"}, "invalid",
			true,
			true,
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		if t, err := time.Parse(core.FRIENDLY_TIME_F, args.localCreated); err == nil {
			args.localPkg.Created = core.YamlTime{t}
		}
		if t, err := time.Parse(core.FRIENDLY_TIME_F, args.remoteCreated); err == nil {
			args.remotePkg.Created = core.YamlTime{t}
		}

		// This is what we're testing here.
		res, err := NeedsUpdate(&args.localPkg, &args.remotePkg, args.compareCreated)

		// Expectations.
		c.Check(err, IsNil)
		c.Check(res, Equals, args.expectedNeedsUpdate)
	}
}
