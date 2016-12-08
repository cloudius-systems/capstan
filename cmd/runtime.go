package cmd

import (
	"fmt"
	"github.com/cloudius-systems/capstan/runtime"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func RuntimePreview(runtimeName string, plain bool) error {
	// Resolve runtime
	rt, err := runtime.PickRuntime(runtime.RuntimeType(runtimeName))
	if err != nil {
		return err
	}

	res := fmt.Sprintln("--------- meta/run.yaml ---------")
	res += fmt.Sprint(composeConf(rt))
	res += fmt.Sprintln("---------------------------------")

	if plain {
		res = removeComments(res)
	}

	// Actually print.
	fmt.Print(res)

	fmt.Println("Use 'capstan runtime init' to persist this template into $CWD/meta/run.yaml")

	return nil
}

func RuntimeInit(runtimeName string, plain bool, force bool) error {
	// Resolve runtime
	rt, err := runtime.PickRuntime(runtime.RuntimeType(runtimeName))
	if err != nil {
		return err
	}

	// Package must exist before we set run command for it.
	if _, err = os.Stat("meta/package.yaml"); os.IsNotExist(err) {
		return fmt.Errorf("Not a valid capstan pakage. Are you in corret directory?\n%s", err)
	}

	// Don't override existing meta/run.yaml
	if _, err = os.Stat("./meta/run.yaml"); err == nil && !force {
		return fmt.Errorf("meta/run.yaml already exists, use --force to override it")
	}

	// Compose content
	content := fmt.Sprint(composeConf(rt))

	if plain {
		content = removeComments(content)
	}

	// Write
	if err = ioutil.WriteFile("meta/run.yaml", []byte(content), 0644); err != nil {
		return fmt.Errorf("Faile to write to meta/run.yaml: %s", err)
	}

	fmt.Println("meta/run.yaml stub successfully added to your package. Please customize it in editor.")

	return nil
}

func RuntimeList() error {
	fmt.Printf("%-20s%-50s%-20s\n", "RUNTIME", "DESCRIPTION", "DEPENDENCIES")
	for _, runtimeType := range runtime.SupportedRuntimes {
		rt, _ := runtime.PickRuntime(runtimeType)
		fmt.Printf("%-20s%-50s%-20s\n", string(runtimeType), rt.GetRuntimeDescription(), rt.GetDependencies())
	}
	return nil
}

func removeComments(s string) string {
	// Remove all comments.
	re := regexp.MustCompile("(?m)^ *" + "#" + ".*$[\r\n]+")
	s = re.ReplaceAllString(s, "")

	// Remove all empty lines.
	re = regexp.MustCompile("(?m)^ *$[\r\n]+")
	s = re.ReplaceAllString(s, "")

	return s
}

func composeConf(rt runtime.Runtime) string {
	res := `
runtime: RUNTIME

config_set: 

   ################################################################
   ### This is one configuration set (feel free to rename it).  ###
   ################################################################
   myconfig1:
      PLACEHOLDER   

   # Add as many named configurations as you need

# OPTIONAL
# What config_set should be used as default.
# This value can be overwritten with --runconfig argument.
config_set_default: myconfig1
`
	// Properly indent runtime-specific part.
	s := strings.TrimSpace(rt.GetYamlTemplate())
	s = strings.Replace(s, "\n", "\n      ", -1)
	res = strings.Replace(res, "PLACEHOLDER", s, -1)

	// Set runtime
	res = strings.Replace(res, "RUNTIME", rt.GetRuntimeName(), -1)
	return res
}
