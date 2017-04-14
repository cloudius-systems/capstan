/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core_test

import (
	"testing"

	"github.com/mikelangelo-project/capstan/core"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type testingCapstanignoreSuite struct{}

var _ = Suite(&testingCapstanignoreSuite{})

func (s *testingCapstanignoreSuite) TestIsIgnored(c *C) {
	m := []struct {
		comment      string
		pattern      string
		path         string
		shouldIgnore bool
	}{
		{
			"fully specified file in root #1",
			"/myfile.txt", "/myfile.txt", true,
		},
		{
			"fully specified file in root #2",
			"/myfile.txt", "/myfolder/myfile.txt", false,
		},

		{
			"fully specified file not in root #1",
			"/myfolder/myfile.txt", "/myfile.txt", false,
		},
		{
			"fully specified file not in root #2",
			"/myfolder/myfile.txt", "/myfolder/myfile.txt", true,
		},
		{
			"file by extension in root #1",
			"/*.txt", "/myfile.txt", true,
		},
		{
			"file by extension in root #2",
			"/*.txt", "/myfolder/myfile.txt", false,
		},
		{
			"file by extension not in root #1",
			"/myfolder/*.txt", "/myfile.txt", false,
		},
		{
			"file by extension not in root #2",
			"/myfolder/*.txt", "/myfolder/myfile.txt", true,
		},
		{
			"file by extension not in root #3",
			"/myfolder/*.txt", "/myfolder/subfolder/myfile.txt", false,
		},
		{
			"fully specified file in any subfolder #1",
			"/**/file.txt", "/myfile.txt", true,
		},
		{
			"fully specified file in any subfolder #2",
			"/**/file.txt", "/myfolder/myfile.txt", true,
		},
		{
			"fully specified file in any subfolder #3",
			"/**/file.txt", "/myfolder/subfolder/myfile.txt", true,
		},
		{
			"whole folder one level #1",
			"/myfolder/*", "/myfolder/myfile.txt", true,
		},
		{
			"whole folder one level #2",
			"/myfolder/*", "/myfolder", true,
		},
		{
			"whole folder one level #3",
			"/myfolder/*", "/myfolder/subfolder/myfile.txt", true,
		},
		{
			"whole folder one level #4",
			"/myfolder/*", "/myfolder/subfolder", true,
		},
		{
			"whole folder two levels #1",
			"/myfolder/subfolder/*", "/myfolder/subfolder", true,
		},
		{
			"whole folder two levels #2",
			"/myfolder/subfolder/*", "/myfolder", false,
		},
		{
			"whole folder two whole levels #1",
			"/myfolder/*/*", "/myfolder", true,
		},
		{
			"whole folder two whole levels #2",
			"/myfolder/*/*", "/myfolder/subfolder", true,
		},
		{
			"whole folder two whole levels #3",
			"/myfolder/*/*", "/myfolder/subfolder/myfile.txt", true,
		},
		{
			"any text file in project #1",
			"/**/*.txt", "/myfile.txt", true,
		},
		{
			"any text file in project #2",
			"/**/*.txt", "/myfolder/myfile.txt", true,
		},
		{
			"any text file in project #3",
			"/**/*.txt", "/myfolder/subfolder/myfile.txt", true,
		},
		{
			"additional test #1",
			"/subfolder/*", "/myfolder/subfolder/myfile.txt", false,
		},
		{
			"additional test #2",
			"/myfolder/*.txt", "/myfolder/myfileXtxt", false,
		},
		{
			"additional test #3",
			"/myfolder", "/myfolder2", false,
		},
		{
			"additional test #4",
			"/myfolder/*", "/myfolder2", false,
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Setup
		capstanignore := core.CapstanignoreInit("", false)
		capstanignore.AddPattern(args.pattern)

		// This is what we're testing here.
		ignoreYesNo := capstanignore.IsIgnored(args.path)

		// Expectations.
		c.Check(ignoreYesNo, Equals, args.shouldIgnore)
	}
}

func (s *testingCapstanignoreSuite) TestIsIgnoredAlways(c *C) {
	paths := map[string]bool{
		"/meta":              true,
		"/meta/package.yml":  true,
		"/meta/":             true,
		"/mpm-pkg":           true,
		"/mpm-pkg/":          true,
		"/mpm-pkg/file":      true,
		"/.git":              true,
		"/.git/":             true,
		"/.git/subpath":      true,
		"/met":               false,
		"/mpm":               false,
		"/.gi":               false,
		"/long/path/to/file": false,
	}

	for path, ignored := range paths {
		capstanignore := core.CapstanignoreInit("", false)
		c.Assert(capstanignore.IsIgnored(path), Equals, ignored)
	}
}
