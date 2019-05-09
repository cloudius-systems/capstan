/*
 * Copyright (C) 2018 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	. "github.com/cloudius-systems/capstan/testing"
	. "gopkg.in/check.v1"
)

type runtimeSuite struct{}

var _ = Suite(&runtimeSuite{})

func (s *suite) TestRuntimeList(c *C) {
	// This is what we're testing here.
	txt := RuntimeList()

	// Expectations.
	expected := `
		RUNTIME {13}DESCRIPTION                             {11}DEPENDENCIES {8}
		native  {13}Run arbitrary command inside OSv        {11}\[\]
		node    {13}Run JavaScript NodeJS 4.4.5 application {11}\[node-4.4.5          \]
		java    {13}Run Java application                    {11}\[openjdk8-zulu-compact1\]
		python  {13}Run Python 2.7 application              {11}\[python-2.7          \]
	`
	c.Check(txt, MatchesMultiline, FixIndent(expected))
}
