package cmd_test

import (
	"bytes"
	"github.com/cloudius-systems/capstan/cmd"
	"github.com/cloudius-systems/capstan/core"
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
	packageYaml := "name: Capstan tester\nauthor: MIKE\nversion: 0.23-24-gc60331d\n" +
		"require:\n - httpserver\n - openmpi\n" +
		"binary:\n app: /usr/bin/app.so\n /usr/bin/app: /usr/local/bin/app.so"

	var simplePackage core.Package
	err := simplePackage.Parse([]byte(packageYaml))

	c.Assert(err, IsNil)
	c.Assert(simplePackage.Name, Equals, "Capstan tester")
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
	minimalYaml := "name: MIKE test\nauthor: MIKE"
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

	err := cmd.ComposePackage(tmp)

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

	err = cmd.ComposePackage(tmp)
	c.Assert(err, NotNil)
}

func (*suite) TestCollectPackageContents(c *C) {
	// We are going to create an empty temp directory.
	tmp, _ := ioutil.TempDir("", "pkg")
	defer os.RemoveAll(tmp)

	// Create package metadata
	metaPath := filepath.Join(tmp, "meta")
	os.MkdirAll(metaPath, 0755)

	simplePacakge := "name: simple\nauthor: mike\nversion: 0.1"

	err := ioutil.WriteFile(filepath.Join(metaPath, "package.yaml"), []byte(simplePacakge), 0644)
	c.Assert(err, IsNil)

	// Also add few files.
	os.MkdirAll(filepath.Join(tmp, "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(tmp, "usr", "lib"), 0755)
	ioutil.WriteFile(filepath.Join(tmp, "file1"), []byte("file1"), 0644)
	ioutil.WriteFile(filepath.Join(tmp, "usr", "bin", "file2"), []byte("file2"), 0644)
	ioutil.WriteFile(filepath.Join(tmp, "usr", "bin", "file3"), []byte("file3"), 0644)
	ioutil.WriteFile(filepath.Join(tmp, "usr", "lib", "file4"), []byte("file4"), 0644)
	ioutil.WriteFile(filepath.Join(tmp, "usr", "lib", "file5"), []byte("file5"), 0644)

	m := make(map[string]string)
	err = cmd.CollectPackageContents(m, tmp)
	c.Assert(err, IsNil)

	c.Assert(len(m), Equals, 8)
	c.Assert(m[filepath.Join(tmp, "file1")], Equals, "/file1")
	c.Assert(m[filepath.Join(tmp, "usr", "bin", "file2")], Equals, "/usr/bin/file2")

	err = cmd.ComposePackage(tmp)
	c.Assert(err, IsNil)
}
