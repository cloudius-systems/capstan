package hypervisor

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type suite struct{}

var _ = Suite(&suite{})

func (*suite) TestParseVolume(c *C) {
	m := []struct {
		comment      string
		volumeString string
		expected     *Volume
	}{
		{
			"simplest",
			"/path/to/volume.img",
			&Volume{
				Path:    "/path/to/volume.img",
				Format:  "raw",
				AioType: "native",
				Cache:   "none",
			},
		},
		{
			"format qcow2",
			"/path/to/volume.img:format=qcow2",
			&Volume{
				Path:    "/path/to/volume.img",
				Format:  "qcow2",
				AioType: "native",
				Cache:   "none",
			},
		},
		{
			"aio threads",
			"/path/to/volume.img:aio=threads",
			&Volume{
				Path:    "/path/to/volume.img",
				Format:  "raw",
				AioType: "threads",
				Cache:   "none",
			},
		},
		{
			"cache writethrough",
			"/path/to/volume.img:cache=writethrough",
			&Volume{
				Path:    "/path/to/volume.img",
				Format:  "raw",
				AioType: "native",
				Cache:   "writethrough",
			},
		},
		{
			"three at a time",
			"/path/to/volume.img:aio=threads:cache=writethrough:format=qcow2",
			&Volume{
				Path:    "/path/to/volume.img",
				Format:  "qcow2",
				AioType: "threads",
				Cache:   "writethrough",
			},
		},
		{
			"uppercase key",
			"/path/to/volume.img:FORMAT=qcow2",
			&Volume{
				Path:    "/path/to/volume.img",
				Format:  "qcow2",
				AioType: "native",
				Cache:   "none",
			},
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// This is what we're testing here.
		volume, err := parseVolume(args.volumeString)

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(volume, DeepEquals, args.expected)
	}
}

func (*suite) TestParseVolumes(c *C) {
	m := []struct {
		comment       string
		volumeStrings []string
		expected      []string
	}{
		{
			"single volume",
			[]string{"/volume1.img"},
			[]string{"/volume1.img"},
		},
		{
			"three volumes, order is preserved",
			[]string{"/volume1.img", "/volume2.img", "/volume3.img"},
			[]string{"/volume1.img", "/volume2.img", "/volume3.img"},
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// This is what we're testing here.
		volumes, err := ParseVolumes(args.volumeStrings)

		// Expectations.
		c.Assert(err, IsNil)
		paths := []string{}
		for _, volume := range volumes {
			paths = append(paths, volume.Path)
		}
		c.Check(paths, DeepEquals, args.expected)
	}
}

func (*suite) TestParseVolumeInvalid(c *C) {
	m := []struct {
		comment      string
		volumeString string
		err          string
	}{
		{
			"only colon",
			"/path/to/volume.img:",
			"Please use '=' for assignment of volume settings. Example: --volume /vol.img:format=raw",
		},
		{
			"only key",
			"/path/to/volume.img:format",
			"Please use '=' for assignment of volume settings. Example: --volume /vol.img:format=raw",
		},
		{
			"illegal attribute",
			"/path/to/volume.img:format=qcow2:illegal=value",
			"Unknown volume setting: 'illegal'",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// This is what we're testing here.
		_, err := parseVolume(args.volumeString)

		// Expectations.
		c.Check(err, ErrorMatches, args.err)
	}
}
