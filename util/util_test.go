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

func (*utilSuite) TestVersionStringToInt(c *C) {
	m := []struct {
		comment  string
		version  string
		expected int
	}{
		{
			"regular case",
			"4.12.10",
			4012010,
		},
		{
			"major version",
			"15.0.0",
			15000000,
		},
		{
			"minor version",
			"0.15.0",
			15000,
		},
		{
			"patch",
			"0.0.15",
			15,
		},
		{
			"missing patch",
			"0.15",
			15000,
		},
		{
			"missing minor",
			"15",
			15000000,
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// This is what we're testing here.
		res, err := VersionStringToInt(args.version)

		// Expectations.
		c.Check(err, IsNil)
		c.Check(res, Equals, args.expected)
	}
}

func (*utilSuite) TestVersionStringToIntInvalid(c *C) {
	m := []struct {
		comment     string
		version     string
		expectedErr string
	}{
		{
			"completely wrong version",
			"x",
			"Invalid version string: 'x'",
		},
		{
			"partially wrong version",
			"1.x.3",
			"Invalid version string: '1.x.3'",
		},
		{
			"empty version",
			"",
			"Invalid version string: ''",
		},
		{
			"continues after patch",
			"1.2.3.4",
			"Invalid version string: '1.2.3.4'",
		},
		{
			"major greater than 999",
			"1000.2.3",
			"Invalid version string: '1000.2.3'",
		},
		{
			"minor greater than 999",
			"1.2000.3",
			"Invalid version string: '1.2000.3'",
		},
		{
			"patch greater than 999",
			"1.2.3000",
			"Invalid version string: '1.2.3000'",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// This is what we're testing here.
		_, err := VersionStringToInt(args.version)

		// Expectations.
		c.Check(err, ErrorMatches, args.expectedErr)
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
