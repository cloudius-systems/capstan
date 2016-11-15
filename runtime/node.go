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
	return []string{"eu.mikelangelo-project.app.node-4.4.5"}
}
func (conf nodeJsRuntime) Validate() error {
	if conf.Main == "" {
		return fmt.Errorf("'main' must be provided")
	}

	return conf.CommonRuntime.Validate()
}
func (conf nodeJsRuntime) GetRunConfig() (*RunConfig, error) {
	return &RunConfig{
		Cmd: fmt.Sprintf("node %s", conf.Main),
	}, nil
}
func (conf nodeJsRuntime) OnCollect(targetPath string) error {
	return nil
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
