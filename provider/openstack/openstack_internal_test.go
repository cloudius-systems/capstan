/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package openstack

import (
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TestingOpenstackSuite struct{}

var _ = Suite(&TestingOpenstackSuite{})

func (s *TestingOpenstackSuite) TestPickOptimalFlavor(c *C) {
	m := []struct {
		comment string
		flavors []flavors.Flavor
		optimal string
		err     string
	}{
		{
			"no matching flavor",
			[]flavors.Flavor{}, "", "No matching flavors to pick from",
		},
		{
			"prefer smaller disk for equal memory",
			[]flavors.Flavor{
				flavors.Flavor{Disk: 30, RAM: 1, ID: "f01"},
				flavors.Flavor{Disk: 10, RAM: 1, ID: "f02"},
				flavors.Flavor{Disk: 20, RAM: 1, ID: "f03"},
				flavors.Flavor{Disk: 40, RAM: 1, ID: "f04"},
			}, "f02", "",
		},
		{
			"prefer smaller disk regardless memory",
			[]flavors.Flavor{
				flavors.Flavor{Disk: 50, RAM: 1, ID: "f01"},
				flavors.Flavor{Disk: 1, RAM: 16, ID: "f02"},
				flavors.Flavor{Disk: 40, RAM: 1, ID: "f03"},
				flavors.Flavor{Disk: 20, RAM: 1, ID: "f04"},
			}, "f02", "",
		},
		{
			"prefer smaller memory for equal hdd",
			[]flavors.Flavor{
				flavors.Flavor{Disk: 10, RAM: 3, ID: "f01"},
				flavors.Flavor{Disk: 10, RAM: 2, ID: "f02"},
				flavors.Flavor{Disk: 10, RAM: 1, ID: "f03"},
				flavors.Flavor{Disk: 10, RAM: 8, ID: "f04"},
			}, "f03", "",
		},
	}
	for _, args := range m {
		c.Log(args.comment)

		flavor, err := selectBestFlavor(args.flavors, false)
		if args.err != "" {
			c.Check(err, ErrorMatches, args.err)
		} else {
			c.Check(flavor, NotNil)
			c.Check(err, IsNil)
			c.Check(flavor.ID, Equals, args.optimal)
		}
	}
}
