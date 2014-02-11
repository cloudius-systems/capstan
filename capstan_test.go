package capstan

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func capstan(command ...string) *exec.Cmd {
	name, _ := ioutil.TempDir("", "capstan-root")
	c := exec.Command("capstan", command...)
	c.Env = append(os.Environ(), fmt.Sprintf("CAPSTAN_ROOT=%s", name))
	return c
}

func TestImages(t *testing.T) {
	cmd := capstan("images")
	out, err := cmd.Output()
	if err != nil {
		t.Errorf("capstan: %v", err)
	}
	if g, e := string(out), ""; g != e {
		t.Errorf("capstan: want %q, got %q", e, g)
	}
}
