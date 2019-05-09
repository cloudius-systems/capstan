/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util_test

import (
	"testing"

	"github.com/cloudius-systems/capstan/util"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type testingParserSuite struct{}

var _ = Suite(&testingParserSuite{})

func (s *testingParserSuite) TestParseEnvironmentList(c *C) {
	m := []struct {
		comment     string
		envList     []string
		expectedRes map[string]string
		err         string
	}{
		{
			"no parameters",
			[]string{},
			map[string]string{},
			"",
		},
		{
			"single parameter",
			[]string{"PORT=8000"},
			map[string]string{"PORT": "8000"},
			"",
		},
		{
			"two parameters",
			[]string{"PORT=8000", "ENDPOINT=foo.com"},
			map[string]string{"PORT": "8000", "ENDPOINT": "foo.com"},
			"",
		},
		{
			"invalid char (space) #1",
			[]string{"NAME=my name"},
			map[string]string{},
			"failed to parse --env argument .*",
		},
		{
			"invalid char (space) #2",
			[]string{"MY NAME=name"},
			map[string]string{},
			"failed to parse --env argument .*",
		},
		{
			"invalid char (space) #3",
			[]string{"MY NAME=my name"},
			map[string]string{},
			"failed to parse --env argument .*",
		},
		{
			"value with equals sign #1",
			[]string{"NAME=my=name"},
			map[string]string{"NAME": "my=name"},
			"",
		},
		{
			"value with equals sign #2",
			[]string{"NAME==name"},
			map[string]string{"NAME": "=name"},
			"",
		},
		{
			"one parameter ok, other not",
			[]string{"PORT=8000", "ENDPOINT=i am invalid"},
			map[string]string{},
			"failed to parse --env argument .*",
		},
		{
			"same parameter two times",
			[]string{"PORT=8000", "PORT=9999"},
			map[string]string{"PORT": "9999"},
			"",
		},
		{
			"empty value",
			[]string{"PORT="},
			map[string]string{"PORT": ""},
			"",
		},
		{
			"not key=value format",
			[]string{"PORT"},
			map[string]string{},
			"failed to parse --env argument .*",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// This is what we're testing here.
		res, err := util.ParseEnvironmentList(args.envList)

		// Expectations.
		if args.err != "" {
			c.Check(err, ErrorMatches, args.err)
		} else {
			c.Check(res, DeepEquals, args.expectedRes)
		}
	}
}
