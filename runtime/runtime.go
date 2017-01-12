package runtime

import (
	"fmt"
	"github.com/mikelangelo-project/capstan/nat"
	"strings"
)

type RuntimeType string

const (
	Native RuntimeType = "native"
	NodeJS RuntimeType = "node"
	Java   RuntimeType = "java"
)

var SupportedRuntimes []RuntimeType = []RuntimeType{
	Native,
	NodeJS,
	Java,
}

type RunConfig struct {
	InstanceName string
	ImageName    string
	Hypervisor   string
	Verbose      bool
	Memory       string
	Cpus         int
	Networking   string
	Bridge       string
	NatRules     []nat.Rule
	GCEUploadDir string
	MAC          string
	Cmd          string
	Persist      bool
}

// Runtime interface must be extended for every new runtime.
// Typically, a runtime struct contains fileds that are expected in
// meta/run.yaml and implements the functions required by this interface.
type Runtime interface {
	// Validate values that were read from yaml.
	Validate() error

	// GetBootCmd produces bootcmd based on meta/run.yaml.
	GetBootCmd() (string, error)

	// GetRuntimeName returns unique runtime name
	// (use constant from the SupportedRuntimes list)
	GetRuntimeName() string

	// GetRuntimeDescription provides short description about what
	// is this runtime used for, 50 chars
	GetRuntimeDescription() string

	// OnCollect is a callback to run when collecting package
	// (accepts directroy path of the package)
	OnCollect(string) error

	// GetYamlTemplate provides a string containing yaml content with
	// as much help text as possible.
	// NOTE: provide only runtime-specific part of yaml, see runtime/node.go for example.
	// NOTE: Write each comment in its own line for --plain flag to remove it.
	GetYamlTemplate() string

	// GetDependencies returns a list of dependent package names.
	GetDependencies() []string

	// GetEnv returns map of environment variables read from run.yaml.
	GetEnv() map[string]string
}

// CommonRuntime fields are those common to all runtimes.
// This fields are set for each named-configuration separately, nothing
// is shared.
type CommonRuntime struct {
	Env map[string]string `yaml:"env"`
}

func (r CommonRuntime) GetEnv() map[string]string {
	return r.Env
}

func (r CommonRuntime) GetYamlTemplate() string {
	return `
# OPTIONAL
# Environment variables.
# A map of environment variables to be set when unikernel is run.
# Example value:  env:
#                    PORT: 8000
#                    HOSTNAME: www.myserver.org
env:
   <key>: <value>
`
}

func (r CommonRuntime) Validate() error {
	for k, v := range r.Env {
		if strings.Contains(k, " ") || strings.Contains(v, " ") {
			return fmt.Errorf("spaces not allowed in env key/value: '%s':'%s'", k, v)
		}
	}
	return nil
}

// BuildBootCmd equips runtime-specific bootcmd with common parts.
func (r CommonRuntime) BuildBootCmd(bootCmd string) (string, error) {
	// Prepend environment variables
	newBootCmd, err := PrependEnvsPrefix(bootCmd, r.GetEnv())
	if err != nil {
		return "", err
	}

	return newBootCmd, nil
}

// PickRuntime maps runtime name into runtime struct.
func PickRuntime(runtimeName RuntimeType) (Runtime, error) {
	switch runtimeName {
	case Native:
		return &nativeRuntime{}, nil
	case NodeJS:
		return &nodeJsRuntime{}, nil
	case Java:
		return &javaRuntime{}, nil
	}

	return nil, fmt.Errorf("Unknown runtime: '%s'\n", runtimeName)
}

// PrependEnvsPrefix prepends all key-values of env map to the boot cmd give.
// It prepends each pair in a form of "--env={KEY}={VALUE}".
// Also performs check that neither key nor value contains space.
func PrependEnvsPrefix(cmd string, env map[string]string) (string, error) {
	s := ""
	for k, v := range env {
		s += fmt.Sprintf("--env=%s=%s ", k, v)
	}
	return fmt.Sprintf("%s%s", s, cmd), nil
}

// BootCmdForScript returns boot command that is to be used
// to run config set with name bootName.
func BootCmdForScript(bootName string) string {
	if bootName == "" {
		return ""
	}

	return fmt.Sprintf("runscript /run/%s", bootName)
}
