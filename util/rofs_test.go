/*
 * Copyright (C) 2018 Waldemar Kozaczuk.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util_test

import (
	"github.com/mikelangelo-project/capstan/cmd"
	"github.com/mikelangelo-project/capstan/util"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type rofsSuite struct{}

var _ = Suite(&rofsSuite{})

func (*rofsSuite) TestWriteRofsImage(c *C) {
	// We are going to create an empty temp directory.
	tmp, _ := ioutil.TempDir("", "pkg")
	defer os.RemoveAll(tmp)
	//
	// Copy test data to the temp dir so that we can create ROFS image out of it
	err := copyDirectory("../cmd/testdata/hashing", tmp)

	paths, err := cmd.CollectDirectoryContents(tmp)
	c.Assert(err, IsNil)

	rofsImagePath := path.Join(tmp, "rofs.img")
	err = util.WriteRofsImage(rofsImagePath, paths, tmp, true)
	c.Assert(err, IsNil)

	rofsImage, err := os.OpenFile(rofsImagePath, os.O_RDONLY, 0644)
	c.Assert(err, IsNil)
	defer rofsImage.Close()

	rofsSb, err := util.ReadRofsSuperBlock(rofsImage)
	c.Assert(err, IsNil)
	c.Assert(rofsSb.InodesCount, Equals, uint64(11))
	c.Assert(rofsSb.DirectoryEntriesCount, Equals, uint64(10))
	c.Assert(rofsSb.SymlinksCount, Equals, uint64(1))
}

func copyDirectory(srcDir string, dest string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, _ error) error {
		relPath := strings.TrimPrefix(path, srcDir)
		fi, err := os.Lstat(path)
		if err != nil {
			return err
		}

		switch {
		case fi.Mode()&os.ModeSymlink == os.ModeSymlink:
			linkTarget, _ := os.Readlink(path)
			if strings.HasPrefix(linkTarget, "/") || strings.HasPrefix(linkTarget, "..") {
				srcDir := filepath.Dir(path)

				if linkTarget, err = filepath.Abs(filepath.Join(srcDir, linkTarget)); err != nil {
					return err
				}
				linkTarget = strings.TrimPrefix(linkTarget, strings.TrimSuffix(path, dest))
			}
			os.Symlink(linkTarget, filepath.Join(dest, relPath))

		case fi.Mode().IsRegular():
			from, err := os.Open(path)
			if err != nil {
				return err
			}
			defer from.Close()

			destFile := filepath.Join(dest, relPath)
			if err := os.MkdirAll(filepath.Dir(destFile), 0777); err != nil {
				return err
			}

			to, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				return err
			}
			defer to.Close()

			if _, err = io.Copy(to, from); err != nil {
				return err
			}
		}
		return nil
	})
}
