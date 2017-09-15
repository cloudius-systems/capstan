/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package testing

import (
	"strings"
)

const TIMESTAMP_REGEX string = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?([+-]\d{2}:\d{2})?[Zz]?`

// FixIndent moves the inline yaml content to the very left.
// This way we are able to write inline yaml content that is
// nicely aligned with other code.
func FixIndent(s string) string {
	s = strings.TrimSpace(s) + "\n"
	return strings.Replace(s, "\t", "", -1)
}
