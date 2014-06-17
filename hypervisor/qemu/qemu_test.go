/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package qemu

import (
	"testing"
)

var parsingtests = []struct {
	in  string
	out *Version
}{
	{"QEMU emulator version 1.6.2, Copyright (c) 2003-2008 Fabrice Bellard", &Version{Major: 1, Minor: 6, Patch: 2}},
	{"QEMU PC emulator version 0.12.1 (qemu-kvm-0.12.1.2), Copyright (c) 2003-2008 Fabrice Bellard", &Version{Major: 0, Minor: 12, Patch: 1}},
}

func TestVersionParsing(t *testing.T) {
	for i, tt := range parsingtests {
		version, err := ParseVersion(tt.in)
		if err != nil {
			t.Errorf("%d. ParseVersion(%q) => error %q, want %q", i, tt.in, err, tt.out)
		}
		if version.Major != tt.out.Major || version.Minor != tt.out.Minor || version.Patch != tt.out.Patch {
			t.Errorf("%d. ParseVersion(%q) => %q, want %q", i, tt.in, version, tt.out)
		}
	}
}
