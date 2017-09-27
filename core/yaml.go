/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core

import (
	"time"
)

const FRIENDLY_TIME_F = "2006-01-02 15:04"

type YamlTime struct {
	Time interface{}
}

// MarshalYAML transforms YamlTime object into a RFC3339 string.
func (t YamlTime) MarshalYAML() (interface{}, error) {
	if v, ok := t.Time.(time.Time); ok {
		return v.Format(time.RFC3339), nil
	} else {
		return "N/A", nil
	}
}

// UnmarshalYAML parses string into YamlTime object. Following formats
// are supported: RFC3339, FRIENDLY_TIME_F
// If time string is invalid, a special YamlTime is created that is then
// marked as '?' when printed as string.
func (t *YamlTime) UnmarshalYAML(unmarshaller func(interface{}) error) error {
	unmarshaller(&t.Time)
	s, _ := t.Time.(string)

	if v, err := time.Parse(time.RFC3339, s); err == nil {
		t.Time = v
	} else if v, err := time.Parse(FRIENDLY_TIME_F, s); err == nil {
		t.Time = v
	} else {
		t.Time = nil
	}

	return nil
}

func (t YamlTime) String() string {
	if v, ok := t.Time.(time.Time); ok {
		return v.Format(FRIENDLY_TIME_F)
	} else {
		return "N/A"
	}
}
