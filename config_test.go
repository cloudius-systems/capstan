package capstan

import (
	"github.com/kylelemons/go-gypsy/yaml"
	"testing"
)

var configTests = []struct {
	Spec string
	Err  string
}{
	{"base: osv-base\n", "yaml: cmdline: \"cmdline\" not found"},
	{"base: osv-base\ncmdline: foo.so\n", ""},
	{"base: osv-base\ncmdline: foo.so\nfiles:\n", ""},
	{"base: osv-base\ncmdline: foo.so\nbuild: make\n", ""},
}

func TestConfig(t *testing.T) {
	for _, test := range configTests {
		_, err := ParseConfig(yaml.Config(test.Spec))
		var got string
		switch err {
		case nil:
			got = ""
		default:
			got = err.Error()
		}
		if want := test.Err; got != want {
			t.Errorf("Get(%q) error %#q, want %#q", test.Spec, got, want)
		}
	}
}
