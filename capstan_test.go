package capstan

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func capstan(command []string) *exec.Cmd {
	name, _ := ioutil.TempDir("", "capstan-root")
	c := exec.Command("capstan", command...)
	c.Env = append(os.Environ(), fmt.Sprintf("CAPSTAN_ROOT=%s", name))
	return c
}

func TestCommandErrors(t *testing.T) {
	m := map[string]string{
		"build foo": "open Capstanfile: no such file or directory\n",
		"build":     "usage: capstan build [image-name]\n",
		"pull":      "usage: capstan pull [image-name]\n",
		"push":      "usage: capstan push [image-name] [image-file]\n",
		"push foo":  "usage: capstan push [image-name] [image-file]\n",
		"rmi":       "usage: capstan rmi [image-name]\n",
		"run foo":   "foo: no such image\n",
		"run":       "usage: capstan run [image-name]\n",
	}
	for key, value := range m {
		cmd := capstan(strings.Fields(key))
		out, err := cmd.Output()
		if err != nil {
			t.Errorf("capstan: %v", err)
		}
		if g, e := string(out), value; g != e {
			t.Errorf("capstan: want %q, got %q", e, g)
		}
	}
}
