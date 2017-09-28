/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime

import (
	"fmt"
	"strings"
)

type pythonRuntime struct {
	CommonRuntime `yaml:"-,inline"`
	PythonArgs    []string `yaml:"python_args"`
	Main          string   `yaml:"main"`
	Args          []string `yaml:"args"`
	IsShell       bool     `yaml:"shell"` // run interactive python interpreter
}

//
// Interface implementation
//

func (conf pythonRuntime) GetRuntimeName() string {
	return string(Python)
}
func (conf pythonRuntime) GetRuntimeDescription() string {
	return "Run Python 2.7 application"
}
func (conf pythonRuntime) GetDependencies() []string {
	return []string{"python-2.7"}
}
func (conf pythonRuntime) Validate() error {
	if conf.Base != "" {
		if conf.IsShell || len(conf.PythonArgs) > 0 || conf.Main != "" || len(conf.Args) > 0 {
			return fmt.Errorf("incompatible arguments specified [shell,python_args,main,args] for custom 'base'")
		}
	} else if conf.IsShell {
		if conf.Main != "" || len(conf.Args) > 0 {
			return fmt.Errorf("incompatible arguments specified [main,args] for shell=true")
		}
		if conf.Env["MAIN"] != "" || conf.Env["ARGS"] != "" {
			return fmt.Errorf("incompatible 'env' keys specified [MAIN,ARGS] for shell=true")
		}
	} else {
		if conf.Main == "" {
			return fmt.Errorf("'main' must be provided")
		}
	}

	return conf.CommonRuntime.Validate()
}
func (conf pythonRuntime) GetBootCmd(cmdConfs map[string]*CmdConfig, env map[string]string) (string, error) {
	conf.Base = "python-2.7:python"
	conf.setDefaultEnv(map[string]string{
		"PYTHON_ARGS": conf.concatPythonArgs(),
	})

	if conf.IsShell {
		conf.Env["MAIN"] = "-"
		conf.Env["ARGS"] = ""
	} else {
		conf.setDefaultEnv(map[string]string{
			"MAIN": conf.Main,
			"ARGS": strings.Join(conf.Args, " "),
		})
	}
	return conf.CommonRuntime.BuildBootCmd("", cmdConfs, env)
}
func (conf pythonRuntime) GetYamlTemplate() string {
	return `
# REQUIRED
# Filepath of the Python script.
# Note that package root will correspond to filesystem root (/) in OSv image.
# Example value: /hello-world.py
main: <filepath>

# OPTIONAL
# A list of Python args.
# Example value: node_args:
#                   - -O
python_args:
   - <list>

# OPTIONAL
# A list of command line args used by the application.
# Example value: args:
#                   - argument1
#                   - argument2
args:
   - <list>
` + conf.CommonRuntime.GetYamlTemplate()
}

//
// Utility
//

func (conf pythonRuntime) concatPythonArgs() string {
	if len(conf.PythonArgs) > 0 {
		return strings.Join(conf.PythonArgs, " ")
	} else {
		// This is a workaround since runscript is currently unable to
		// handle empty environment variable as a parameter. So we set
		// dummy value unless user provided some actual value.
		return "-O"
	}
}
