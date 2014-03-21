/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package nat

import (
	"strings"
)

type Rule struct {
	HostPort  string
	GuestPort string
}

func Parse(rules []string) []Rule {
	fwds := make([]Rule, 0, 0)
	for _, rule := range rules {
		ports := strings.Split(rule, ":")
		fwds = append(fwds, Rule{HostPort: ports[0], GuestPort: ports[1]})
	}
	return fwds
}
