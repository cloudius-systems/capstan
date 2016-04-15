package cmd

import (
	"bytes"
	"github.com/cloudius-systems/capstan/core"
	"github.com/cloudius-systems/capstan/util"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type suite struct {
	capstanBinary string
}

func (s *suite) SetUpSuite(c *C) {
	s.capstanBinary, _ = filepath.Abs("../capstan")
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

func (s *suite) TestPackageCommandExists(c *C) {
	cmd := exec.Command(s.capstanBinary, "package", "help", "compose")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(out.String(), "No help topic"), Equals, false)
}

func (*suite) TestComposeNonPackageFails(c *C) {
	// We are going to create an empty temp directory.
	tmp, _ := ioutil.TempDir("", "pkg")
	defer os.RemoveAll(tmp)

	repo := util.NewRepo(util.DefaultRepositoryUrl)
	imageSize, _ := util.ParseMemSize("64M")
	appName := "test-app"

	err := ComposePackage(repo, imageSize, false, false, tmp, appName)

	c.Assert(err, NotNil)
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
	appName := "test-app"

	err = ComposePackage(repo, imageSize, false, false, tmp, appName)
	c.Assert(err, NotNil)
}

func (*suite) TestCollectDirectoryContents(c *C) {
	paths, err := collectDirectoryContents("testdata/hashing")
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
