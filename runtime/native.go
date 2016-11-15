package runtime

import (
	"fmt"
)

type nativeRuntime struct {
	CommonRuntime `yaml:"-,inline"`
	BootCmd       string `yaml:"bootcmd"`
}

//
// Interface implementation
//

func (conf nativeRuntime) GetRuntimeName() string {
	return string(Native)
}
func (conf nativeRuntime) GetRuntimeDescription() string {
	return "Run arbitrary command inside OSv"
}
func (conf nativeRuntime) GetDependencies() []string {
	return []string{}
}
func (conf nativeRuntime) Validate() error {
	if conf.BootCmd == "" {
		return fmt.Errorf("'bootcmd' must be provided")
	}

	return conf.CommonRuntime.Validate()
}
func (conf nativeRuntime) GetRunConfig() (*RunConfig, error) {
	return &RunConfig{
		Cmd: fmt.Sprintf("%s", conf.BootCmd),
	}, nil
}
func (conf nativeRuntime) OnCollect(targetPath string) error {
	return nil
}
func (conf nativeRuntime) GetYamlTemplate() string {
	return `
# REQUIRED
# Command to be executed in OSv.
# Note that package root will correspond to filesystem root (/) in OSv image.
# Example value: /usr/bin/simpleFoam.so -help
bootcmd: <command>
`
}
