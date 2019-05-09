/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 * Modifications copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util_test

import (
	"fmt"
	"path/filepath"

	"github.com/cloudius-systems/capstan/cmd"
	"github.com/cloudius-systems/capstan/util"

	. "github.com/cloudius-systems/capstan/testing"
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

func (s *suite) TestImageList(c *C) {
	m := []struct {
		comment   string
		imagePath string
		indexYaml string
		expected  string
	}{
		{
			"usual case",
			"mike/myimage",
			`
				description: description
				format_version: 1
				version: 9aba80a
				created: 2017-08-02 08:16
				platform: Ubuntu-14.04
			`,
			`
				Name         {39}Description {40}Version {9}Created          {5}Platform
				mike/myimage {39}description {40}9aba80a {9}2017-08-02 08:16 {5}Ubuntu-14.04
			`,
		},
		{
			"missing fields",
			"mike/myimage",
			`
				description: description
				format_version: 1
			`,
			`
				Name         {39}Description {40}Version {9}Created {14}Platform
				mike/myimage {39}description {40}        {9}N/A     {14}N/A
			`,
		},
		{
			"invalid index.yaml",
			"mike/myimage",
			`
				xyz
			`,
			`
				Name         {39}Description {40}Version {9}Created {14}Platform
				mike/myimage {39}            {40}        {9}N/A     {14}N/A
			`,
		},
		{
			"missing index.yaml",
			"mike/myimage",
			"",
			`
				Name         {39}Description {40}Version {9}Created {14}Platform
				mike/myimage {39}            {40}        {9}N/A     {14}N/A
			`,
		},
		{
			"usual case (no namespace)",
			"myimage",
			`
				description: description
				format_version: 1
				version: 9aba80a
				created: 2017-08-02 08:16
				platform: Ubuntu-14.04
			`,
			`
				Name    {44}Description {40}Version {9}Created          {5}Platform
				myimage {44}description {40}9aba80a {9}2017-08-02 08:16 {5}Ubuntu-14.04
			`,
		},
		{
			"missing index.yaml (no namespace)",
			"myimage",
			"",
			`
				Name    {44}Description {40}Version {9}Created {14}Platform
				myimage {44}            {40}        {9}N/A     {14}N/A
			`,
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		ClearDirectory(s.repo.Path)
		files := map[string]string{
			fmt.Sprintf("repository/%s/myimage.qemu", args.imagePath): DefaultText,
		}
		if args.indexYaml != "" {
			files[fmt.Sprintf("repository/%s/index.yaml", args.imagePath)] = FixIndent(args.indexYaml)
		}
		PrepareFiles(s.repo.Path, files)

		// This is what we're testing here.
		txt := s.repo.ListImages()

		// Expectations.
		c.Check(txt, MatchesMultiline, FixIndent(args.expected))
	}
}

func (s *suite) TestLocalPackages(c *C) {
	m := []struct {
		comment          string
		search           string
		packages         []string
		clutterFiles     map[string]string
		expectedPackages []string
	}{
		{
			"no packages",
			"",
			[]string{},
			map[string]string{},
			[]string{},
		},
		{
			"single package",
			"",
			[]string{"package1"},
			map[string]string{},
			[]string{"package1"},
		},
		{
			"single package - ignore clutter",
			"",
			[]string{"package1"},
			map[string]string{
				"clutter.txt": DefaultText,
			},
			[]string{"package1"},
		},
		{
			"two packages",
			"",
			[]string{"package1", "package2"},
			map[string]string{},
			[]string{"package1", "package2"},
		},
		{
			"search",
			"package1",
			[]string{"package1", "package2"},
			map[string]string{},
			[]string{"package1"},
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		ClearDirectory(s.repo.Path)
		for _, pkgName := range args.packages {
			s.importPkg(map[string]string{
				"meta/package.yaml": FixIndent(fmt.Sprintf(`
					name: %s
					title: title
					author: author
			`, pkgName)),
			}, c)
		}
		PrepareFiles(s.repo.PackagesPath(), args.clutterFiles)

		// This is what we're testing here.
		packages, err := s.repo.LocalPackages(args.search)

		// Expectations.
		c.Check(err, IsNil)
		names := []string{}
		for _, pkg := range packages {
			names = append(names, pkg.Name)
		}
		c.Check(names, DeepEquals, args.expectedPackages)
	}
}

func (s *suite) TestLocalPackagesInvalid(c *C) {
	m := []struct {
		comment      string
		clutterFiles map[string]string
		expectedErr  string
	}{
		{
			"invalid package",
			map[string]string{
				"invalid-package.yaml": DefaultText,
			},
			"invalid package manifest",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		ClearDirectory(s.repo.Path)
		PrepareFiles(s.repo.PackagesPath(), args.clutterFiles)

		// This is what we're testing here.
		_, err := s.repo.LocalPackages("")

		// Expectations.
		c.Assert(err, NotNil)
		c.Check(err.Error(), MatchesMultiline, args.expectedErr)
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
