/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudius-systems/capstan/core"
	"github.com/cloudius-systems/capstan/util"

	. "github.com/cloudius-systems/capstan/testing"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type suite struct {
	capstanBinary string
	packageDir    string
	packageFiles  map[string]string
	repo          *util.Repo
}

func (s *suite) SetUpSuite(c *C) {
	s.capstanBinary, _ = filepath.Abs("../capstan")
}

func (s *suite) SetUpTest(c *C) {
	s.packageDir = c.MkDir()
	s.packageFiles = map[string]string{
		"/meta/package.yaml":  PackageYamlText,
		"/file.txt":           DefaultText,
		"/data/data-file.txt": DefaultText,
		"/meta/README.md":     DefaultText,
	}
	PrepareFiles(s.packageDir, s.packageFiles)
	s.repo = util.NewRepo(util.DefaultRepositoryUrl)
	s.repo.Path = c.MkDir()
}

var _ = Suite(&suite{})

func (*suite) TestPackageUnmarshaller(c *C) {
	packageYaml := "name: Capstan tester\ntitle: MPM Test package\nauthor: MIKE\nversion: 0.23-24-gc60331d\n" +
		"require:\n - httpserver\n - openmpi\n" +
		"binary:\n app: /usr/bin/app.so\n /usr/bin/app: /usr/local/bin/app.so"

	var simplePackage core.Package
	err := simplePackage.Parse([]byte(packageYaml))

	c.Assert(err, IsNil)
	c.Assert(simplePackage.Name, Equals, "Capstan tester")
	c.Assert(simplePackage.Title, Equals, "MPM Test package")
	c.Assert(simplePackage.Author, Equals, "MIKE")
	c.Assert(simplePackage.Version, Equals, "0.23-24-gc60331d")
	c.Assert(simplePackage.Require, HasLen, 2)
	c.Assert(simplePackage.Binary["app"], Equals, "/usr/bin/app.so")
	c.Assert(simplePackage.Binary["/usr/bin/app"], Equals, "/usr/local/bin/app.so")
}

func (*suite) TestInvalidYaml(c *C) {
	packageYaml := "name Capstan tester"

	var pkg core.Package
	err := pkg.Parse([]byte(packageYaml))

	c.Assert(err, NotNil)
}

func (*suite) TestIncomplete(c *C) {
	emptyYaml := ""

	var emptyPackage core.Package
	err := emptyPackage.Parse([]byte(emptyYaml))

	c.Assert(err, NotNil)

	nameYaml := "name: MIKE test"
	var namePackage core.Package
	err = namePackage.Parse([]byte(nameYaml))
	c.Assert(err, NotNil)
}

func (*suite) TestMinimalPackageYaml(c *C) {
	minimalYaml := "name: MIKE test\ntitle: MIKELANGELO test package\nauthor: MIKE"
	var nameAuthorPackage core.Package
	err := nameAuthorPackage.Parse([]byte(minimalYaml))
	c.Assert(err, IsNil)
}

func (s *suite) TestInitPackage(c *C) {
	m := []struct {
		comment         string
		pkg             core.Package
		expectedPkgYaml string
	}{
		{
			"simplest case",
			core.Package{
				Name:   "name",
				Title:  "title",
				Author: "author",
			},
			`
				name: name
				title: title
				author: author
				created: "{TIMESTAMP}"
			`,
		},
		{
			"with version",
			core.Package{
				Name:    "name",
				Title:   "title",
				Author:  "author",
				Version: "1.2.3",
			},
			`
				name: name
				title: title
				author: author
				version: 1.2.3
				created: "{TIMESTAMP}"
			`,
		},
		{
			"with require",
			core.Package{
				Name:    "name",
				Title:   "title",
				Author:  "author",
				Require: []string{"demo1", "demo2"},
			},
			`
				name: name
				title: title
				author: author
				require:
				- demo1
				- demo2
				created: "{TIMESTAMP}"
			`,
		},
		{
			"with created",
			core.Package{
				Name:    "name",
				Title:   "title",
				Author:  "author",
				Created: core.YamlTime{time.Now()},
			},
			`
				name: name
				title: title
				author: author
				created: "{TIMESTAMP}"
			`,
		},
		{
			"with platform",
			core.Package{
				Name:     "name",
				Title:    "title",
				Author:   "author",
				Platform: "Ubuntu-14.04",
			},
			`
				name: name
				title: title
				author: author
				created: "{TIMESTAMP}"
				platform: Ubuntu-14.04
			`,
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		expected := FixIndent(strings.Replace(args.expectedPkgYaml, "{TIMESTAMP}", TIMESTAMP_REGEX, -1))

		// This is what we're testing here.
		err := InitPackage(s.packageDir, &args.pkg)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(filepath.Join(s.packageDir, "meta", "package.yaml"), FileMatches, expected)
	}
}

func (*suite) TestComposeWithNoManifestSucceeds(c *C) {
	// We are going to create an empty temp directory.
	tmp, _ := ioutil.TempDir("", "pkg")
	defer os.RemoveAll(tmp)

	repo := util.NewRepo(util.DefaultRepositoryUrl)
	imageSize, _ := util.ParseMemSize("64M")
	appName := "test-corrupt-app"

	err := ComposePackage(repo, []string {}, imageSize, false, false, true, tmp, appName, &BootOptions{}, "rofs")

	c.Assert(err, IsNil)
}

func (*suite) TestComposeCorruptPackageFails(c *C) {
	// We are going to create an empty temp directory.
	tmp, _ := ioutil.TempDir("", "pkg")
	defer os.RemoveAll(tmp)

	// Create package metadata
	metaPath := filepath.Join(tmp, "meta")
	os.MkdirAll(metaPath, 0755)

	err := ioutil.WriteFile(filepath.Join(metaPath, "package.yaml"), []byte("illegal package"), 0644)
	c.Assert(err, IsNil)

	repo := util.NewRepo(util.DefaultRepositoryUrl)
	imageSize, _ := util.ParseMemSize("64M")
	appName := "test-corrupt-app"

	err = ComposePackage(repo, []string {}, imageSize, false, false, false, tmp, appName, &BootOptions{}, "zfs")
	c.Assert(err, NotNil)
}

func (*suite) TestCollectDirectoryContents(c *C) {
	paths, err := CollectDirectoryContents("testdata/hashing")
	c.Assert(err, IsNil)

	expectedPaths := []string{"file1", "symlink-to-file1", "dir2", "dir2/file-in-dir2", "dir1",
		"dir1/file2", "dir1/dir3", "dir1/dir3/another-file", "dir1/dir3/file3", "file4"}

	c.Assert(paths, HasLen, len(expectedPaths))

	wd, err := os.Getwd()
	if err != nil {
		c.Fail()
	}

	for _, path := range expectedPaths {
		hostPath := filepath.Join(wd, "testdata", "hashing", path)
		guestPath := filepath.Join("/", path)

		c.Assert(paths[hostPath], Equals, guestPath)
	}
}

func (*suite) TestFileHashing(c *C) {
	expectedHashes := map[string]string{
		"/file1":                  "5235be9b9e4ae0c8f4a7037b122cdec4",
		"/symlink-to-file1":       "5235be9b9e4ae0c8f4a7037b122cdec4",
		"/file4":                  "d41d8cd98f00b204e9800998ecf8427e",
		"/dir2/file-in-dir2":      "bab32b2dd8c64c63af1214a1bebd59d8",
		"/dir1/file2":             "cabe46f8749fde430f75df84c82a433a",
		"/dir1/dir3/another-file": "b2a63c3b7990c175a2bd03bc6f35397e",
		"/dir1/dir3/file3":        "65b17cb1d1308e8bead96db0f31125b5",
		"/dir1":                   "fd4470862b13f32bfcc3659aa8dc4082",
		"/dir1/dir3":              "fa983bf68e65476b95e362f3d1ff3cf2",
	}

	wd, err := os.Getwd()
	if err != nil {
		c.Fail()
	}

	for path, hash := range expectedHashes {
		hostPath := filepath.Join(wd, "testdata", "hashing", path)

		hostHash, err := hashPath(hostPath, path)
		c.Assert(err, IsNil)

		c.Assert(hostHash, Equals, hash)
	}
}

func (s *suite) TestBuildPackage(c *C) {
	// This is what we're testing here.
	resultFile, err := BuildPackage(s.packageDir)

	// Expectations.
	c.Assert(err, IsNil)
	c.Check(resultFile, Equals, filepath.Join(s.packageDir, "package-name.mpm"))
	expectedFiles := map[string]interface{}{
		"/meta/package.yaml":  PackageYamlText,
		"/file.txt":           DefaultText,
		"/data/data-file.txt": DefaultText,
		"/meta/README.md":     DefaultText,
	}
	c.Check(resultFile, TarGzEquals, expectedFiles)
}

func (s *suite) TestDescribePackage(c *C) {
	// Prepare
	ImportPackage(s.repo, s.packageDir)

	// This is what we're testing here.
	descr, err := DescribePackage(s.repo, "package-name")

	// Expectations.
	c.Assert(err, IsNil)
	c.Check(descr, MatchesMultiline, fmt.Sprintf(".*PACKAGE DOCUMENTATION\n%s\n", DefaultText))
}

func (s *suite) TestRecursiveRunYamls(c *C) {
	// Prepare.
	s.importFakeOSvBootstrapPkg(c)
	s.importFakeDemoPkg(c)
	s.requireFakeDemoPkg(c)

	// This is what we're testing here.
	err := CollectPackage(s.repo, s.packageDir, []string {}, false, false, false)

	// Expectations.
	c.Assert(err, IsNil)
	expectedBoots := map[string]interface{}{
		"demoBoot1": "echo Demo1",
		"demoBoot2": "echo Demo2",
	}
	c.Check(filepath.Join(s.packageDir, "mpm-pkg", "run"), DirEquals, expectedBoots)
}

func (s *suite) TestRecursiveRunYamlsWithOwnRunYaml(c *C) {
	// Prepare.
	s.importFakeOSvBootstrapPkg(c)
	s.importFakeDemoPkg(c)
	s.requireFakeDemoPkg(c)
	s.setRunYaml(`
		runtime: native
		config_set:
		  ownBoot:
		    bootcmd: echo MyBoot
	`, c)

	// This is what we're testing here.
	err := CollectPackage(s.repo, s.packageDir, []string {},false, false, false)

	// Expectations.
	c.Assert(err, IsNil)
	expectedBoots := map[string]interface{}{
		"demoBoot1": "echo Demo1",
		"demoBoot2": "echo Demo2",
		"ownBoot":   "echo MyBoot",
	}
	c.Check(filepath.Join(s.packageDir, "mpm-pkg", "run"), DirEquals, expectedBoots)
}

func (s *suite) TestRecursiveRunYamlsWithOwnRunYamlOverwrite(c *C) {
	// Prepare.
	s.importFakeOSvBootstrapPkg(c)
	s.importFakeDemoPkg(c)
	s.requireFakeDemoPkg(c)
	s.setRunYaml(`
		runtime: native
		config_set:
		  demoBoot1:
		    bootcmd: echo MyBoot
	`, c)

	// This is what we're testing here.
	err := CollectPackage(s.repo, s.packageDir, []string {},false, false, false)

	// Expectations.
	c.Assert(err, IsNil)
	expectedBoots := map[string]interface{}{
		"demoBoot1": "echo MyBoot",
		"demoBoot2": "echo Demo2",
	}
	c.Check(filepath.Join(s.packageDir, "mpm-pkg", "run"), DirEquals, expectedBoots)
}

func (s *suite) TestRecursiveRunYamlsWithOwnRunYamlEnv(c *C) {
	// Prepare.
	s.importFakeOSvBootstrapPkg(c)
	s.importFakeDemoPkg(c)
	s.requireFakeDemoPkg(c)
	s.setRunYaml(`
		runtime: native
		config_set:
		  demoBoot1:
		    bootcmd: echo MyBoot
		    env:
		      PORT: 8000
		      HOST: localhost
		  demoBoot2:
		    bootcmd: echo MyBoot2
		    env:
		      PORT: 3000
	`, c)

	// This is what we're testing here.
	err := CollectPackage(s.repo, s.packageDir, []string {}, false, false, false)

	// Expectations.
	c.Assert(err, IsNil)
	expectedBoots := map[string]interface{}{
		"demoBoot1": checkBootCmd("echo MyBoot", []string{"--env=PORT?=8000", "--env=HOST?=localhost"}),
		"demoBoot2": checkBootCmd("echo MyBoot2", []string{"--env=PORT?=3000"}),
	}
	c.Check(filepath.Join(s.packageDir, "mpm-pkg", "run"), DirEquals, expectedBoots)
}

func (s *suite) TestAbsTarPathMatches(c *C) {
	m := []struct {
		comment     string
		tarPath     string
		pathPattern string
		shouldMatch bool
	}{
		{
			"absolute pattern #1",
			"/meta/run.yaml", "/meta/run.yaml", true,
		},
		{
			"absolute pattern #2",
			"meta/run.yaml", "/meta/run.yaml", true,
		},
		{
			"absolute pattern #3",
			"mydir/meta/run.yaml", "/meta/run.yaml", false,
		},
		{
			"relative pattern #1",
			"/meta/run.yaml", "meta/run.yaml", true,
		},
		{
			"relative pattern #2",
			"meta/run.yaml", "meta/run.yaml", true,
		},
		{
			"relative pattern #3",
			"mydir/meta/run.yaml", "meta/run.yaml", false,
		},
		{
			"all in dir",
			"/meta/run.yaml", "/meta/.*", true,
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// This is what we're testing here.
		match := absTarPathMatches(args.tarPath, args.pathPattern)

		// Expectations.
		c.Check(match, Equals, args.shouldMatch)
	}
}

func (s *suite) TestRuntimeInheritance(c *C) {
	// Prepare.
	s.importFakeOSvBootstrapPkg(c)

	m := []struct {
		comment         string
		runYamlText     string
		demoRunYamlText string
		expected        map[string]interface{}
	}{
		{
			"basic",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:demoBoot1"
			`,
			"",
			map[string]interface{}{
				"demoBoot1": "echo Demo1",
				"demoBoot2": "echo Demo2",
				"ownBoot":   "echo Demo1",
			},
		},
		{
			"with own env",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:demoBoot1"
				    env:
				      PORT: 8000
			`,
			"",
			map[string]interface{}{
				"demoBoot1": "echo Demo1",
				"demoBoot2": "echo Demo2",
				"ownBoot":   "--env=PORT?=8000 echo Demo1",
			},
		},
		{
			"with own env forced",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:demoBoot1"
				    env:
				      PORT: 8000
			`,
			`
				runtime: native
				config_set:
				  demoBoot1:
				    bootcmd: echo Demo1
				    env:
				      PORT: 1111
			`,
			map[string]interface{}{
				"demoBoot1": "--env=PORT?=1111 echo Demo1",
				"ownBoot":   "--env=PORT?=8000 echo Demo1",
			},
		},
		{
			"with own env forced doesn't interfere",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:demoBoot1"
				    env:
				      PORT: 8000
				  ownBoot2:
				    base: "fake.demo:demoBoot1"
			`,
			`
				runtime: native
				config_set:
				  demoBoot1:
				    bootcmd: echo Demo1
				    env:
				      PORT: 1111
			`,
			map[string]interface{}{
				"demoBoot1": "--env=PORT?=1111 echo Demo1",
				"ownBoot":   "--env=PORT?=8000 echo Demo1",
				"ownBoot2":  "--env=PORT?=1111 echo Demo1",
			},
		},
		{
			"with own env forced combined",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:demoBoot1"
				    env:
				      PORT: 8000
			`,
			`
				runtime: native
				config_set:
				  demoBoot1:
				    bootcmd: echo Demo1
				    env:
				      PORT: 1111
				      HOST: localhost
			`,
			map[string]interface{}{
				"demoBoot1": checkBootCmd("echo Demo1", []string{"--env=HOST?=localhost", "--env=PORT?=1111"}),
				"ownBoot":   checkBootCmd("echo Demo1", []string{"--env=HOST?=localhost", "--env=PORT?=8000"}),
			},
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)
		s.requireFakeDemoPkg(c)
		// Prepare
		if args.demoRunYamlText != "" {
			s.importFakeDemoPkgWithRunYaml(args.demoRunYamlText, c)
		} else {
			s.importFakeDemoPkg(c)
		}
		s.setRunYaml(args.runYamlText, c)

		// This is what we're testing here.
		err := CollectPackage(s.repo, s.packageDir, []string {}, false, false, false)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(filepath.Join(s.packageDir, "mpm-pkg", "run"), DirEquals, args.expected)
	}
}

func (s *suite) TestRuntimeInheritInvalid(c *C) {
	// Prepare.
	s.importFakeOSvBootstrapPkg(c)
	s.importFakeDemoPkg(c)
	s.requireFakeDemoPkg(c)

	m := []struct {
		comment     string
		runYamlText string
		error       string
	}{
		{
			"invalid package",
			`
			runtime: native
			config_set:
			  ownBoot:
			    base: "unknown:demoBoot1"
			`,
			"Failed to inherit from 'unknown': package not included or has no meta/run.yaml",
		},
		{
			"invalid config_set",
			`
			runtime: native
			config_set:
			  ownBoot:
			    base: "fake.demo:unknown"
			`,
			"Failed to inherit 'fake.demo:unknown': config_set does not exist",
		},
		{
			"empty base",
			`
			runtime: native
			config_set:
			  ownBoot:
			    base:
			`,
			"Validation failed for configuration set 'ownBoot': 'bootcmd' must be provided",
		},
		{
			"invalid base #1",
			`
			runtime: native
			config_set:
			  ownBoot:
			    base: ":"
			`,
			"Failed to inherit from '': package not included or has no meta/run.yaml",
		},
		{
			"invalid base #2",
			`
			runtime: native
			config_set:
			  ownBoot:
			    base: "missing-column"
			`,
			"Validation failed for configuration set 'ownBoot': 'base' must be in format <pkg_name>:<config_set>",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare
		s.setRunYaml(args.runYamlText, c)

		// This is what we're testing here.
		err := CollectPackage(s.repo, s.packageDir, []string {},false, false, false)

		// Expectations.
		c.Assert(err, NotNil)
		c.Check(err, ErrorMatches, args.error)
	}
}

func (s *suite) TestRuntimeInheritanceTwoLevels(c *C) {
	// Prepare.
	s.importFakeOSvBootstrapPkg(c)

	m := []struct {
		comment          string
		runYamlText      string
		demoRunYamlText  string
		demo2RunYamlText string
		expected         map[string]interface{}
	}{
		{
			"simple",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:intermediate"
			`,
			`
				runtime: native
				config_set:
				  intermediate:
				    base: "fake.demo2:deepest"
			`,
			`
				runtime: native
				config_set:
				  deepest:
				    bootcmd: echo Deepest
				    env:
				      PORT: 1111
			`,
			map[string]interface{}{
				"ownBoot":      "--env=PORT?=1111 echo Deepest",
				"intermediate": "--env=PORT?=1111 echo Deepest",
				"deepest":      "--env=PORT?=1111 echo Deepest",
			},
		},
		{
			"my env wins #1",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:intermediate"
				    env:
				      PORT: 8000
			`,
			`
				runtime: native
				config_set:
				  intermediate:
				    base: "fake.demo2:deepest"
			`,
			`
				runtime: native
				config_set:
				  deepest:
				    bootcmd: echo Deepest
				    env:
				      PORT: 1111
			`,
			map[string]interface{}{
				"ownBoot":      "--env=PORT?=8000 echo Deepest",
				"intermediate": "--env=PORT?=1111 echo Deepest",
				"deepest":      "--env=PORT?=1111 echo Deepest",
			},
		},
		{
			"my env wins #2",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:intermediate"
				    env:
				      PORT: 8000
			`,
			`
				runtime: native
				config_set:
				  intermediate:
				    base: "fake.demo2:deepest"
				    env:
				      PORT: 1234
			`,
			`
				runtime: native
				config_set:
				  deepest:
				    bootcmd: echo Deepest
				    env:
				      PORT: 1111
			`,
			map[string]interface{}{
				"ownBoot":      "--env=PORT?=8000 echo Deepest",
				"intermediate": "--env=PORT?=1234 echo Deepest",
				"deepest":      "--env=PORT?=1111 echo Deepest",
			},
		},
		{
			"my additional env",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:intermediate"
				    env:
				      PORT: 8000
				      HOST: localhost
			`,
			`
				runtime: native
				config_set:
				  intermediate:
				    base: "fake.demo2:deepest"
				    env:
				      PORT: 1234
			`,
			`
				runtime: native
				config_set:
				  deepest:
				    bootcmd: echo Deepest
				    env:
				      PORT: 1111
			`,
			map[string]interface{}{
				"ownBoot":      checkBootCmd("echo Deepest", []string{"--env=PORT?=8000", "--env=HOST?=localhost"}),
				"intermediate": "--env=PORT?=1234 echo Deepest",
				"deepest":      "--env=PORT?=1111 echo Deepest",
			},
		},
		{
			"intermediate wins",
			`
				runtime: native
				config_set:
				  ownBoot:
				    base: "fake.demo:intermediate"
			`,
			`
				runtime: native
				config_set:
				  intermediate:
				    base: "fake.demo2:deepest"
				    env:
				      PORT: 1234
			`,
			`
				runtime: native
				config_set:
				  deepest:
				    bootcmd: echo Deepest
				    env:
				      PORT: 1111
			`,
			map[string]interface{}{
				"ownBoot":      "--env=PORT?=1234 echo Deepest",
				"intermediate": "--env=PORT?=1234 echo Deepest",
				"deepest":      "--env=PORT?=1111 echo Deepest",
			},
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)
		s.requireFakeDemoPkgOneAndTwo(c)
		// Prepare
		s.importFakeDemoPkgWithRunYaml(args.demoRunYamlText, c)
		s.importFakeDemo2PkgWithRunYaml(args.demo2RunYamlText, c)

		s.setRunYaml(args.runYamlText, c)

		// This is what we're testing here.
		err := CollectPackage(s.repo, s.packageDir, []string {},false, false, false)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(filepath.Join(s.packageDir, "mpm-pkg", "run"), DirEquals, args.expected)
	}
}

func (s *suite) TestGetCmd(c *C) {
	m := []struct {
		comment        string
		options        BootOptions
		confSetDefault string
		expectedCmd    string
		err            string
	}{
		{
			"empty command line",
			BootOptions{},
			"",
			"runscript /run/default;",
			"",
		},
		{
			"simulate --execute",
			BootOptions{Cmd: "direct cmd"},
			"",
			"direct cmd",
			"",
		},
		{
			"simulate --boot",
			BootOptions{Boot: []string{"boot1"}},
			"",
			"runscript /run/boot1;",
			"",
		},
		{
			"simulate multiple --boot",
			BootOptions{Boot: []string{"boot1", "boot2", "boot3"}},
			"",
			"runscript /run/boot1;runscript /run/boot2;runscript /run/boot3;",
			"",
		},
		{
			"simulate config_set_default",
			BootOptions{PackageDir: s.packageDir},
			"default1",
			"runscript /run/default1;",
			"",
		},
		{
			"simulate multiple config_set_default",
			BootOptions{PackageDir: s.packageDir},
			"default1,default2",
			"runscript /run/default1;runscript /run/default2;",
			"",
		},
		{
			"--execute is most important",
			BootOptions{
				Cmd:        "direct cmd",
				Boot:       []string{"boot1"},
				PackageDir: s.packageDir,
			},
			"",
			"direct cmd",
			"",
		},
		{
			"--boot is second most important",
			BootOptions{
				Boot:       []string{"boot1"},
				PackageDir: s.packageDir,
			},
			"",
			"runscript /run/boot1;",
			"",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		if args.confSetDefault != "" {
			s.setRunYaml(fmt.Sprintf(
				`
					runtime: native
					config_set:
					  default:
					    bootcmd: some-invalid-so.so
					config_set_default: %s
				`, args.confSetDefault), c)
		}

		// This is what we're testing here.
		cmd, err := args.options.GetCmd()

		// Expectations.
		if args.err == "" {
			c.Assert(err, IsNil)
		} else {
			c.Assert(err, ErrorMatches, args.err)
		}
		c.Check(cmd, Equals, args.expectedCmd)
	}
}

func (s *suite) TestImplicitlyRequiredPackages(c *C) {
	// Prepare.
	s.importFakeOSvBootstrapPkg(c)
	s.importFakeOSvComposeRemotePkg(c)

	m := []struct {
		comment  string
		remote   bool
		expected map[string]interface{}
	}{
		{
			"collect for local composing",
			false,
			map[string]interface{}{
				"data-file.txt":               DefaultText,
				"osv-bootstrap-data-file.txt": DefaultText, // present in osv.bootstrap
			},
		},
		{
			"collect for remote composing",
			true,
			map[string]interface{}{
				"data-file.txt":                    DefaultText,
				"osv-compose-remote-data-file.txt": DefaultText, // present in osv.compose-remote
			},
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)
		// Prepare

		// This is what we're testing here.
		err := CollectPackage(s.repo, s.packageDir, []string {}, false, args.remote, false)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(filepath.Join(s.packageDir, "mpm-pkg", "data"), DirEquals, args.expected)
	}
}

//
// Utility
//

func (s *suite) importFakeOSvBootstrapPkg(c *C) {
	packageYamlText := FixIndent(`
		name: osv.bootstrap
		title: PackageTitle
		author: package-author
	`)
	files := map[string]string{
		"/meta/package.yaml":                packageYamlText,
		"/meta/README.md":                   DefaultText,
		"/osv-bootstrap-file.txt":           DefaultText,
		"/data/osv-bootstrap-data-file.txt": DefaultText,
	}
	tmpDir := c.MkDir()
	PrepareFiles(tmpDir, files)
	ImportPackage(s.repo, tmpDir)
}

func (s *suite) importFakeOSvComposeRemotePkg(c *C) {
	packageYamlText := FixIndent(`
		name: osv.compose-remote
		title: PackageTitle
		author: package-author
	`)
	files := map[string]string{
		"/meta/package.yaml":                     packageYamlText,
		"/meta/README.md":                        DefaultText,
		"/osv-compose-remote-file.txt":           DefaultText,
		"/data/osv-compose-remote-data-file.txt": DefaultText,
	}
	tmpDir := c.MkDir()
	PrepareFiles(tmpDir, files)
	ImportPackage(s.repo, tmpDir)
}

func (s *suite) importFakeDemoPkg(c *C) {
	packageYamlText := FixIndent(`
		name: fake.demo
		title: Fake Demo
		author: Demo Author
	`)
	runYamlText := FixIndent(`
		runtime: native
		config_set:
		  demoBoot1:
		    bootcmd: echo Demo1
		  demoBoot2:
		    bootcmd: echo Demo2
	`)
	files := map[string]string{
		"/meta/package.yaml":            packageYamlText,
		"/meta/run.yaml":                runYamlText,
		"/meta/README.md":               DefaultText,
		"/fake-demo-file.txt":           DefaultText,
		"/data/fake-demo-data-file.txt": DefaultText,
	}
	s.importPkg(files, c)
}

func (s *suite) importFakeDemoPkgWithRunYaml(runYamlText string, c *C) {
	packageYamlText := FixIndent(`
		name: fake.demo
		title: Fake Demo
		author: Demo Author
	`)

	files := map[string]string{
		"/meta/package.yaml": packageYamlText,
		"/meta/run.yaml":     FixIndent(runYamlText),
	}
	s.importPkg(files, c)
}

func (s *suite) importFakeDemo2PkgWithRunYaml(runYamlText string, c *C) {
	packageYamlText := FixIndent(`
		name: fake.demo2
		title: Fake Demo2
		author: Demo Author2
	`)

	files := map[string]string{
		"/meta/package.yaml": packageYamlText,
		"/meta/run.yaml":     FixIndent(runYamlText),
	}
	s.importPkg(files, c)
}

func (s *suite) importPkg(files map[string]string, c *C) {
	tmpDir := c.MkDir()
	PrepareFiles(tmpDir, files)
	ImportPackage(s.repo, tmpDir)
}

// requireFakeDemoPkg sets such meta/package.yaml to our demo package that it
// requires fake.demo package.
func (s *suite) requireFakeDemoPkg(c *C) {
	packageYamlText := FixIndent(`
		name: package-name
		title: PackageTitle
		author: package-author
		require:
		  - fake.demo
	`)
	ioutil.WriteFile(filepath.Join(s.packageDir, "meta", "package.yaml"), []byte(packageYamlText), 0700)
}

func (s *suite) requireFakeDemoPkgOneAndTwo(c *C) {
	packageYamlText := FixIndent(`
		name: package-name
		title: PackageTitle
		author: package-author
		require:
		  - fake.demo
		  - fake.demo2
	`)
	ioutil.WriteFile(filepath.Join(s.packageDir, "meta", "package.yaml"), []byte(packageYamlText), 0700)
}

// setRunYaml sets given content of meta/run.yaml to our demo package.
func (s *suite) setRunYaml(runYamlText string, c *C) {
	ioutil.WriteFile(filepath.Join(s.packageDir, "meta", "run.yaml"), []byte(FixIndent(runYamlText)), 0700)
}

// checkBootCmd prepares lambda function that can be passed to DirEquals.
func checkBootCmd(bootCmd string, env []string) func(string) error {
	return func(v string) error { return CheckBootCmd(v, bootCmd, env) }
}
