/*
 * Copyright (C) 2018 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"os"

	. "gopkg.in/check.v1"
)

type utilSuite struct{}

func (s *utilSuite) SetUpTest(c *C) {}

var _ = Suite(&utilSuite{})

func (*utilSuite) TestConfigDir(c *C) {
	m := []struct {
		comment  string
		env      map[string]string
		expected string
	}{
		{
			"simplest case",
			map[string]string{
				"HOME": "/my/home",
			},
			"/my/home/.capstan",
		},
		{
			"with CAPSTAN_ROOT",
			map[string]string{
				"HOME":         "/my/home",
				"CAPSTAN_ROOT": "/capstan/root",
			},
			"/capstan/root",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		originalEnv := setEnvironmentVars(args.env)

		// This is what we're testing here.
		dir := ConfigDir()

		// Expectations.
		c.Check(dir, Equals, args.expected)

		// Restore original environemt variables
		setEnvironmentVars(originalEnv)
	}
}

//
// Utility
//

func setEnvironmentVars(env map[string]string) map[string]string {
	original := map[string]string{}
	for key, value := range env {
		original[key] = value
		os.Setenv(key, value)
	}
	return original
}
