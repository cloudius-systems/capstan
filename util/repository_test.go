/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 * Modifications copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util_test

import (
	"github.com/mikelangelo-project/capstan/util"
	. "gopkg.in/check.v1"
	"path/filepath"
)

type suite struct {
	repo *util.Repo
}

func (s *suite) SetUpTest(c *C) {
	s.repo = util.NewRepo(util.DefaultRepositoryUrl)
}

var _ = Suite(&suite{})

func (s *suite) TestImagePath(c *C) {
	path := s.repo.ImagePath("qemu", "valid")
	c.Assert(path, Equals, filepath.Join(util.HomePath(), ".capstan", "repository", "valid", "valid.qemu"))
}

func (s *suite) TestPackagePath(c *C) {
	path := s.repo.PackagePath("package")
	c.Assert(path, Equals, filepath.Join(util.HomePath(), ".capstan", "packages", "package.mpm"))
}
