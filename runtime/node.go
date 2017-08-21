/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime

import (
	"fmt"
)

type nodeJsRuntime struct {
	CommonRuntime `yaml:"-,inline"`
	Main          string `yaml:"main"`
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
	inherit := conf.Base != ""

	if !inherit {
		if conf.Main == "" {
			return fmt.Errorf("'main' must be provided")
		}
	}

	return conf.CommonRuntime.Validate(inherit)
}
func (conf nodeJsRuntime) GetBootCmd(cmdConfs map[string]*CmdConfig, env map[string]string) (string, error) {
	cmd := fmt.Sprintf("node %s", conf.Main)
	return conf.CommonRuntime.BuildBootCmd(cmd, cmdConfs, env)
}
func (conf nodeJsRuntime) GetYamlTemplate() string {
	return `
# REQUIRED
# Filepath of the NodeJS entrypoint (where server is defined).
# Note that package root will correspond to filesystem root (/) in OSv image.
# Example value: /server.js
main: <filepath>
` + conf.CommonRuntime.GetYamlTemplate()
}
