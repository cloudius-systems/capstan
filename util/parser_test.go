/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"testing"
)

func TestParseMemSize(t *testing.T) {
	m := map[string]int64{
		"64MB": 64,
		"64M":  64,
		"64mb": 64,
		"64m":  64,
		"1GB":  1024,
		"1G":   1024,
		"1gb":  1024,
		"1g":   1024,
	}
	for key, value := range m {
		size, err := ParseMemSize(key)
		if err != nil {
			t.Errorf("capstan: %v", err)
		}
		if e, g := value, size; e != g {
			t.Errorf("capstan: want %q, got %q", e, g)
		}
	}
}

func TestParseMemSizeErrors(t *testing.T) {
	m := map[string]string{
		"0M":    "0M: memory size must be larger than zero",
		"0G":    "0G: memory size must be larger than zero",
		"64foo": "64foo: unrecognized memory size",
		"64":    "64: unrecognized memory size",
		"foo":   "foo: unrecognized memory size",
	}
	for key, value := range m {
		size, err := ParseMemSize(key)
		if err == nil {
			t.Errorf("capstan: expected error, got %d", size)
		}
		if err != nil && err.Error() != value {
			t.Errorf("capstan: %v", err)
		}
	}
}
