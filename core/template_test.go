/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core

import (
	"testing"
)

var configTests = []struct {
	Spec string
	Err  string
}{
	{"base: osv-base\n", "\"cmdline\" not found"},
	{"base: osv-base\ncmdline: foo.so\n", ""},
	{"base: osv-base\ncmdline: foo.so\nfiles:\n", ""},
	{"base: osv-base\ncmdline: foo.so\nbuild: make\n", ""},
}

func TestTemplate(t *testing.T) {
	for _, test := range configTests {
		_, err := ParseTemplate([]byte(test.Spec))
		var got string
		switch err {
		case nil:
			got = ""
		default:
			got = err.Error()
		}
		if want := test.Err; got != want {
			t.Errorf("Get(%q) error %#q, want %#q", test.Spec, got, want)
		}
	}
}
