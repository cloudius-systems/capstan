package runtime

import (
	"fmt"
)

type nodeJsRuntimeSettings struct {
	Main string `yaml:"main"`
}

//
// Interface implementation
//

func (conf nodeJsRuntimeSettings) GetRuntimeName() string {
	return NodeJS
}
func (conf nodeJsRuntimeSettings) GetRuntimeDescription() string {
	return "Run JavaScript NodeJS 4.4.5 application"
}
func (conf nodeJsRuntimeSettings) GetDependencies() []string {
	return []string{"eu.mikelangelo-project.app.node-4.4.5"}
}
func (conf nodeJsRuntimeSettings) Validate() error {
	return nil
}
func (conf nodeJsRuntimeSettings) GetRunConfig() (*RunConfig, error) {
	return &RunConfig{
		Cmd: fmt.Sprintf("node %s", conf.Main),
	}, nil
}
func (conf nodeJsRuntimeSettings) OnCollect(targetPath string) error {
	return nil
}
func (conf nodeJsRuntimeSettings) GetYamlTemplate() string {
	return `
# REQUIRED
# Filepath of the NodeJS entrypoint (where server is defined).
# Note that package root will correspond to filesystem root (/) in OSv image.
# Example value: /server.js
main: <filepath>
`
}
