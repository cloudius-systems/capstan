/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 * Modifications copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util_test

import (
	"path/filepath"

	"github.com/mikelangelo-project/capstan/cmd"
	"github.com/mikelangelo-project/capstan/util"

	. "github.com/mikelangelo-project/capstan/testing"
	. "gopkg.in/check.v1"
)

type suite struct {
	repo *util.Repo
}

func (s *suite) SetUpTest(c *C) {
	s.repo = util.NewRepo(util.DefaultRepositoryUrl)
	s.repo.Path = c.MkDir()
}

var _ = Suite(&suite{})

func (s *suite) TestImagePath(c *C) {
	path := s.repo.ImagePath("qemu", "valid")
	c.Assert(path, Equals, filepath.Join(s.repo.Path, "repository", "valid", "valid.qemu"))
}

func (s *suite) TestPackagePath(c *C) {
	path := s.repo.PackagePath("package")
	c.Assert(path, Equals, filepath.Join(s.repo.Path, "packages", "package.mpm"))
}

func (s *suite) TestPackageList(c *C) {
	m := []struct {
		comment  string
		pkgYaml  string
		expected string
	}{
		{
			"simplest case",
			`
				name: name
				title: description
				author: author
			`,
			`
				Name {47}Description {40}Version {9}Created {14}Platform
				name {47}description {40}        {9}N/A
			`,
		},
		{
			"with version",
			`
				name: name
				title: description
				author: author
				version: 0.1
			`,
			`
				Name {47}Description {40}Version {9}Created {14}Platform
				name {47}description {40}0.1     {9}N/A
			`,
		},
		{
			"with created",
			`
				name: name
				title: description
				author: author
				created: 2017-07-31 14:49
			`,
			`
				Name {47}Description {40}Version {9}Created          {5}Platform
				name {47}description {40}        {9}2017-07-31 14:49
			`,
		},
		{
			"with platform",
			`
				name: name
				title: description
				author: author
				platform: Ubuntu-14.04
			`,
			`
				Name {47}Description {40}Version {9}Created {14}Platform
				name {47}description {40}        {9}N/A     {14}Ubuntu-14.04
			`,
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		files := map[string]string{
			"meta/package.yaml": FixIndent(args.pkgYaml),
		}
		s.importPkg(files, c)

		// This is what we're testing here.
		txt := s.repo.ListPackages()

		// Expectations.
		c.Check(txt, MatchesMultiline, FixIndent(args.expected))
	}
}

//
// Utility
//

func (s *suite) importPkg(files map[string]string, c *C) {
	tmpDir := c.MkDir()
	PrepareFiles(tmpDir, files)
	cmd.ImportPackage(s.repo, tmpDir)
}
