package core

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Java struct {
	Main      string
	Args      []string
	Classpath []string
	VmArgs    []string
}

func (j *Java) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, j); err != nil {
		return err
	}

	if j.Main == "" {
		return fmt.Errorf("'main' must be provided")
	}

	if j.Classpath == nil {
		return fmt.Errorf("'classpath' must be provided")
	}

	return nil
}

func (j *Java) GetCommandLine() string {
	var cp, args string

	if len(j.Classpath) > 0 {
		cp = "-cp " + strings.Join(j.Classpath, ":")
	}

	if len(j.Args) > 0 {
		args = strings.Join(j.Args, " ")
	}

	return strings.TrimSpace(fmt.Sprintf("%s %s %s", cp, j.Main, args))
}

func (j *Java) GetVmArgs() string {
	vmargs := ""

	for _, arg := range j.VmArgs {
		vmargs += fmt.Sprintf("-%s ", arg)
	}

	return strings.TrimSpace(vmargs)
}

func ParseJavaConfig(packageDir string) (*Java, error) {
	var java *Java

	javaConfig := filepath.Join(packageDir, "meta", "java.yaml")
	if _, err := os.Stat(javaConfig); os.IsNotExist(err) {
		return nil, err
	}

	data, err := ioutil.ReadFile(javaConfig)
	if err != nil {
		return nil, err
	}

	java = &Java{}
	err = java.Parse(data)

	return java, err
}

func IsJavaPackage(packageDir string) bool {
	// We only have to look for java config file.
	javaConfig := filepath.Join(packageDir, "meta", "java.yaml")

	_, err := os.Stat(javaConfig)
	return err == nil
}
