/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime_test

import (
	"testing"

	"github.com/cloudius-systems/capstan/runtime"
	. "github.com/cloudius-systems/capstan/testing"
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
		expectedEnv []string
		err         string
	}{
		{
			"no variable in environment",
			"/node server.js", map[string]string{}, false,
			"/node server.js", []string{},
			"",
		},
		{
			"single variable in environment",
			"/node server.js", map[string]string{"PORT": "8000"}, false,
			"/node server.js", []string{"--env=PORT=8000"},
			"",
		},
		{
			"two variables in environment",
			"/node server.js", map[string]string{"PORT": "8000", "ENDPOINT": "foo.com"}, false,
			"/node server.js", []string{"--env=PORT=8000", "--env=ENDPOINT=foo.com"},
			"",
		},
		{
			"no variable in environment - soft",
			"/node server.js", map[string]string{}, true,
			"/node server.js", []string{},
			"",
		},
		{
			"single variable in environment - soft",
			"/node server.js", map[string]string{"PORT": "8000"}, true,
			"/node server.js", []string{"--env=PORT?=8000"},
			"",
		},
		{
			"two variables in environment - soft",
			"/node server.js", map[string]string{"PORT": "8000", "ENDPOINT": "foo.com"}, true,
			"/node server.js", []string{"--env=PORT?=8000", "--env=ENDPOINT?=foo.com"},
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
			c.Check(res, BootCmdEquals, args.expectedCmd, args.expectedEnv)
		}
	}
}
