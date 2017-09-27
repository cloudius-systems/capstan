/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core

import (
	"testing"
	"time"

	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v2"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type yamlSuite struct {
}

var _ = Suite(&yamlSuite{})

type yamlWithTimeFieldStruct struct {
	Created YamlTime `yaml:"created"`
}

func (*yamlSuite) TestYamlTimeFieldMarshall(c *C) {

	m := []struct {
		comment  string
		created  string
		expected string
	}{
		{
			"simple",
			"2017-09-14T18:08:16+02:00",
			"created: 2017-09-14T18:08:16+02:00",
		},
		{
			"miliseconds should not get marshalled",
			"2017-09-14T18:08:16.123456789+02:00",
			"created: 2017-09-14T18:08:16+02:00",
		},
		{
			"empty",
			"",
			"created: N/A",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare
		yamlTime := YamlTime{}
		if t, _ := time.Parse(time.RFC3339, args.created); args.created != "" {
			yamlTime.Time = t
		}
		obj := yamlWithTimeFieldStruct{
			Created: yamlTime,
		}

		// This is what we're testing here.
		data, err := yaml.Marshal(&obj)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(string(data), Equals, args.expected+"\n")
	}
}

func (*yamlSuite) TestYamlTimeFieldUnmarshall(c *C) {

	m := []struct {
		comment  string
		yamltext string
		expected string
	}{
		{
			"RFC3339",
			"created: 2017-09-14T18:08:16+02:00",
			"2017-09-14 18:08",
		},
		{
			"RFC3339 with miliseconds",
			"created: 2017-09-14T18:08:16.123456789+02:00",
			"2017-09-14 18:08",
		},
		{
			"friendly",
			"created: 2017-09-14 18:08",
			"2017-09-14 18:08",
		},
		{
			"empty",
			"created: ",
			"N/A",
		},
		{
			"missing",
			"",
			"N/A",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare
		obj := yamlWithTimeFieldStruct{}

		// This is what we're testing here.
		err := yaml.Unmarshal([]byte(args.yamltext), &obj)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(obj.Created.String(), Equals, args.expected)
	}
}
