package testing

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type suite struct{}

var _ = Suite(&suite{})

func (*suite) TestContainsArrayStringCheck(c *C) {
	m := []struct {
		comment  string
		obtained []string
		expected []string
		err      string
	}{
		{
			"simplest",
			[]string{"a", "b", "c"},
			[]string{"b", "c"},
			"",
		},
		{
			"full",
			[]string{"a", "b", "c"},
			[]string{"a", "b", "c"},
			"",
		},
		{
			"mismatch #1",
			[]string{"a", "b", "c"},
			[]string{"x"},
			"Obtained array does not contain expected subarray",
		},
		{
			"mismatch #2",
			[]string{"a", "b", "c"},
			[]string{"a", "c"},
			"Obtained array does not contain expected subarray",
		},
		{
			"too short obtained",
			[]string{"a"},
			[]string{"x", "y", "z"},
			"Obtained array is shorter than wanted",
		},
		{
			"empty expected",
			[]string{"a", "b", "c"},
			[]string{},
			"Expected array must not be empty",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		obtained := []interface{}{args.obtained, args.expected}
		names := []string{}

		// This is what we're testing here.
		isOk, errStr := ContainsArray.Check(obtained, names)

		// Expectations.
		c.Assert(errStr, Equals, args.err)
		c.Assert(isOk, Equals, args.err == "")
	}
}

func (*suite) TestContainsArrayIntCheck(c *C) {
	m := []struct {
		comment  string
		obtained []int
		expected []int
		err      string
	}{
		{
			"simplest",
			[]int{1, 2, 3},
			[]int{2, 3},
			"",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		obtained := []interface{}{args.obtained, args.expected}
		names := []string{}

		// This is what we're testing here.
		isOk, errStr := ContainsArray.Check(obtained, names)

		// Expectations.
		c.Assert(errStr, Equals, args.err)
		c.Assert(isOk, Equals, args.err == "")
	}
}
