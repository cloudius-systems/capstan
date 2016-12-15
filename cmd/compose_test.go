package cmd_test

import (
	"github.com/mikelangelo-project/capstan/cmd"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"path/filepath"
)

type TestingComposeSuite struct {
	capstanBinary string
}

func (s *TestingComposeSuite) SetUpSuite(c *C) {
	s.capstanBinary = "../capstan"
}

var _ = Suite(&TestingComposeSuite{})

func (s *TestingComposeSuite) TestCollectPathContentsWithDir(c *C) {
	// We are going to create an empty temp directory.
	tmp, _ := ioutil.TempDir("", "compose")
	defer os.RemoveAll(tmp)

	// Also add few files.
	os.MkdirAll(filepath.Join(tmp, "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(tmp, "usr", "lib"), 0755)
	ioutil.WriteFile(filepath.Join(tmp, "file1"), []byte("file1"), 0644)
	ioutil.WriteFile(filepath.Join(tmp, "usr", "bin", "file2"), []byte("file2"), 0644)
	ioutil.WriteFile(filepath.Join(tmp, "usr", "bin", "file3"), []byte("file3"), 0644)
	ioutil.WriteFile(filepath.Join(tmp, "usr", "lib", "file4"), []byte("file4"), 0644)
	ioutil.WriteFile(filepath.Join(tmp, "usr", "lib", "file5"), []byte("file5"), 0644)

	var m map[string]string
	m, err := cmd.CollectPathContents(tmp)
	c.Assert(err, IsNil)

	c.Assert(len(m), Equals, 8)
	c.Assert(m[filepath.Join(tmp, "file1")], Equals, "/file1")
	c.Assert(m[filepath.Join(tmp, "usr", "bin", "file2")], Equals, "/usr/bin/file2")
}

func (s *TestingComposeSuite) TestCollectPathContentsWithFile(c *C) {
	// We are going to create an empty temp directory.
	tmp, _ := ioutil.TempDir("", "compose")
	//defer os.RemoveAll(tmp)

	// Also add few files.
	os.MkdirAll(filepath.Join(tmp, "usr", "bin"), 0755)
	ioutil.WriteFile(filepath.Join(tmp, "usr", "bin", "file1"), []byte("file1"), 0644)

	var m map[string]string
	m, err := cmd.CollectPathContents(filepath.Join(tmp, "usr", "bin", "file1"))
	c.Assert(err, IsNil)

	c.Assert(len(m), Equals, 1)
	c.Assert(m[filepath.Join(tmp, "usr", "bin", "file1")], Equals, "/file1")
}
