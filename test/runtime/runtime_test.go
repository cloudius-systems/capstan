/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime_test

import (
	"testing"

	"github.com/mikelangelo-project/capstan/runtime"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type testingRuntimeSuite struct{}

var _ = Suite(&testingRuntimeSuite{})

func (s *testingRuntimeSuite) TestPrependEnvsPrefix(c *C) {
	m := []struct {
		comment     string
		cmd         string
		env         map[string]string
		soft        bool
		expectedCmd string
		err         string
	}{
		{
			"no variable in environment",
			"/node server.js",
			map[string]string{},
			false,
			"/node server.js",
			"",
		},
		{
			"single variable in environment",
			"/node server.js",
			map[string]string{"PORT": "8000"},
			false,
			"--env=PORT=8000 /node server.js",
			"",
		},
		{
			"two variables in environment",
			"/node server.js",
			map[string]string{"PORT": "8000", "ENDPOINT": "foo.com"},
			false,
			// Order is not guaranteed hence the following regex is needed:
			"(--env=PORT=8000 --env=ENDPOINT=foo.com|--env=ENDPOINT=foo.com --env=PORT=8000) /node server.js",
			"",
		},
		{
			"no variable in environment - soft",
			"/node server.js",
			map[string]string{},
			true,
			"/node server.js",
			"",
		},
		{
			"single variable in environment - soft",
			"/node server.js",
			map[string]string{"PORT": "8000"},
			true,
			"--env=PORT\\?=8000 /node server.js",
			"",
		},
		{
			"two variables in environment - soft",
			"/node server.js",
			map[string]string{"PORT": "8000", "ENDPOINT": "foo.com"},
			true,
			// Order is not guaranteed hence the following regex is needed:
			"(--env=PORT\\?=8000 --env=ENDPOINT\\?=foo.com|--env=ENDPOINT\\?=foo.com --env=PORT\\?=8000) /node server.js",
			"",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// This is what we're testing here.
		res, err := runtime.PrependEnvsPrefix(args.cmd, args.env, args.soft)

		// Expectations.
		if args.err != "" {
			c.Check(err, ErrorMatches, args.err)
		} else {
			c.Check(res, Matches, args.expectedCmd)
		}
	}
}
