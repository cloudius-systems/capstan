// +build linux darwin

/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"github.com/kylelemons/goat/termios"
)

func RawTerm() (*termios.TermSettings, error) {
	tio, err := termios.NewTermSettings(0)
	if err != nil {
		return nil, err
	}
	err = tio.Raw()
	if err != nil {
		return nil, err
	}
	return tio, err
}

func ResetTerm(tio *termios.TermSettings) {
	tio.Reset()
}
