/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime

import (
	"fmt"
	"strings"
)

type nodeJsRuntime struct {
	CommonRuntime `yaml:"-,inline"`
	NodeArgs      []string `yaml:"node_args"`
	Main          string   `yaml:"main"`
	Args          []string `yaml:"args"`
	IsShell       bool     `yaml:"shell"` // run interactive node interpreter
}

//
// Interface implementation
//

func (conf nodeJsRuntime) GetRuntimeName() string {
	return string(NodeJS)
}
func (conf nodeJsRuntime) GetRuntimeDescription() string {
	return "Run JavaScript NodeJS 4.4.5 application"
}
func (conf nodeJsRuntime) GetDependencies() []string {
	return []string{"node-4.4.5"}
}
func (conf nodeJsRuntime) Validate() error {
	if conf.Base != "" {
		if conf.IsShell || len(conf.NodeArgs) > 0 || conf.Main != "" || len(conf.Args) > 0 {
			return fmt.Errorf("incompatible arguments specified [shell,node_args,main,args] for custom 'base'")
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
func (conf nodeJsRuntime) GetBootCmd(cmdConfs map[string]*CmdConfig, env map[string]string) (string, error) {
	conf.Base = "node-4.4.5:node"
	conf.setDefaultEnv(map[string]string{
		"NODE_ARGS": conf.concatNodeArgs(),
	})

	if conf.IsShell {
		conf.Env["MAIN"] = ""
		conf.Env["ARGS"] = ""
	} else {
		conf.setDefaultEnv(map[string]string{
			"MAIN": conf.Main,
			"ARGS": strings.Join(conf.Args, " "),
		})
	}
	return conf.CommonRuntime.BuildBootCmd("", cmdConfs, env)
}
func (conf nodeJsRuntime) GetYamlTemplate() string {
	return `
# REQUIRED
# Filepath of the NodeJS entrypoint (where server is defined).
# Note that package root will correspond to filesystem root (/) in OSv image.
# Example value: /server.js
main: <filepath>

# OPTIONAL
# A list of Node.js args.
# Example value: node_args:
#                   - --require module1
node_args:
   - <list>

# OPTIONAL
# A list of command line args used by the application.
# Example value: args:
#                   - argument1
#                   - argument2
args:
   - <list>

# OPTIONAL
# Set to true to only run node shell. Note that "main" and "args" will then be ignored.
shell: false
` + conf.CommonRuntime.GetYamlTemplate()
}

//
// Utility
//

func (conf nodeJsRuntime) concatNodeArgs() string {
	if len(conf.NodeArgs) > 0 {
		return strings.Join(conf.NodeArgs, " ")
	} else {
		// This is a workaround since runscript is currently unable to
		// handle empty environment variable as a parameter. So we set
		// dummy value unless user provided some actual value.
		return "--"
	}
}
