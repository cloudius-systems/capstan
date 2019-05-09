/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudius-systems/capstan/hypervisor"
	"github.com/cloudius-systems/capstan/util"

	. "github.com/cloudius-systems/capstan/testing"
	. "gopkg.in/check.v1"
)

type volumesSuite struct {
	capstanBinary string
	packageDir    string
	packageFiles  map[string]string
	repo          *util.Repo
}

func (s *volumesSuite) SetUpTest(c *C) {
	s.packageDir = c.MkDir()
	os.Chmod(s.packageDir, 0777)
}

var _ = Suite(&volumesSuite{})

func (s *volumesSuite) TestCreateVolume(c *C) {
	m := []struct {
		comment      string
		volume       Volume
		expectedMeta string
		expectedInfo string
	}{
		{
			"create default format",
			Volume{
				SizeMB: 128,
				Name:   "vol1",
			},
			`
				format: raw
			`,
			`
				file format: raw
				virtual size: 128M \(134217728 bytes\)
			`,
		},
		{
			"create qcow2",
			Volume{
				Volume: hypervisor.Volume{
					Format: "qcow2",
				},
				SizeMB: 128,
				Name:   "vol1",
			},
			`
				format: qcow2
			`,
			`
				file format: qcow2
				virtual size: 128M \(134217728 bytes\)
			`,
		},
		{
			"create raw",
			Volume{
				Volume: hypervisor.Volume{
					Format: "raw",
				},
				SizeMB: 128,
				Name:   "vol1",
			},
			`
				format: raw
			`,
			`
				file format: raw
				virtual size: 128M \(134217728 bytes\)
			`,
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		volumesDir := filepath.Join(s.packageDir, "volumes")
		metaFile := filepath.Join(volumesDir, fmt.Sprintf("%s.yaml", args.volume.Name))
		volumeFile := filepath.Join(volumesDir, args.volume.Name)
		ClearDirectory(volumesDir)
		PrepareFiles(s.packageDir, map[string]string{"/meta/package.yaml": PackageYamlText})

		// This is what we're testing here.
		err := CreateVolume(s.packageDir, args.volume)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(metaFile, FileMatches, FixIndent(args.expectedMeta))
		c.Check([]string{"qemu-img", "info", volumeFile}, CmdOutputMatches, FixIndent(args.expectedInfo))
	}
}

func (s *volumesSuite) TestCreateVolumeInvalidNoPackage(c *C) {
	// This is what we're testing here.
	err := CreateVolume(s.packageDir, Volume{})

	// Expectations.
	c.Check(err, ErrorMatches, "Must be in package root directory")
}

func (s *volumesSuite) TestCreateVolumeInvalidAlreadyExists(c *C) {
	// Prepare.
	PrepareFiles(s.packageDir, map[string]string{
		"/meta/package.yaml": PackageYamlText,
		"/volumes/vol1":      DefaultText,
		"/volumes/vol1.yaml": DefaultText,
	})

	// This is what we're testing here.
	err := CreateVolume(s.packageDir, Volume{Name: "vol1", Volume: hypervisor.Volume{Format: "vdi"}})

	// Expectations.
	c.Check(err, ErrorMatches, "Could not create volume: Volume already exists")
}

func (s *volumesSuite) TestDeleteVolume(c *C) {
	m := []struct {
		comment         string
		state           map[string]string
		name            string
		expectedVolumes map[string]interface{}
	}{
		{
			"simple",
			map[string]string{
				"/volumes/vol1":      DefaultText,
				"/volumes/vol1.yaml": DefaultText,
			},
			"vol1",
			map[string]interface{}{},
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		volumesDir := filepath.Join(s.packageDir, "volumes")
		args.state["/meta/package.yaml"] = PackageYamlText
		PrepareFiles(s.packageDir, args.state)

		// This is what we're testing here.
		err := DeleteVolume(s.packageDir, args.name, false)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(volumesDir, DirEquals, args.expectedVolumes)
	}
}

func (s *volumesSuite) TestDeleteVolumeInvalid(c *C) {
	m := []struct {
		comment string
		state   map[string]string
		name    string
		err     string
	}{
		{
			"delete when not even volumes directory exists",
			map[string]string{},
			"nonexistent",
			"Could not find volume with name 'nonexistent'",
		},
		{
			"delete nonexistent",
			map[string]string{
				"/volumes/vol1.qcow2":      DefaultText,
				"/volumes/vol1.qcow2.yaml": DefaultText,
			},
			"nonexistent",
			"Could not find volume with name 'nonexistent'",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare.
		args.state["/meta/package.yaml"] = PackageYamlText
		PrepareFiles(s.packageDir, args.state)

		// This is what we're testing here.
		err := DeleteVolume(s.packageDir, args.name, false)

		// Expectations.
		c.Assert(err, ErrorMatches, args.err)
	}
}
