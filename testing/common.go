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

// FixIndent moves the inline yaml content to the very left.
// This way we are able to write inline yaml content that is
// nicely aligned with other code.
func FixIndent(s string) string {
	return strings.Replace(s, "\t", "", -1)
}
