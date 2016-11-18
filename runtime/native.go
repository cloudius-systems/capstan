package runtime

import (
	"fmt"
)

type nativeRuntimeSettings struct {
	Main string `yaml:"main"`
}

//
// Interface implementation
//

func (conf nativeRuntimeSettings) GetRuntimeName() string {
	return Native
}
func (conf nativeRuntimeSettings) GetRuntimeDescription() string {
	return "Run arbitrary command inside OSv"
}
func (conf nativeRuntimeSettings) GetDependencies() []string {
	return []string{}
}
func (conf nativeRuntimeSettings) Validate() error {
	return nil
}
func (conf nativeRuntimeSettings) GetRunConfig() (*RunConfig, error) {
	return &RunConfig{
		Cmd: fmt.Sprintf("%s", conf.Main),
	}, nil
}
func (conf nativeRuntimeSettings) OnCollect(targetPath string) error {
	return nil
}
func (conf nativeRuntimeSettings) GetYamlTemplate() string {
	return `
# REQUIRED
# Command to be executed in OSv.
# Note that package root will correspond to filesystem root (/) in OSv image.
# Example value: --env=WM_PROJECT_DIR=/openfoam /usr/bin/simpleFoam.so -help
main: <command>
`
}
