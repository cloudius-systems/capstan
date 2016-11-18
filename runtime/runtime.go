package runtime

import (
	"fmt"
	"github.com/cloudius-systems/capstan/nat"
)

const (
	Native string = "native"
	NodeJS string = "node"
	Java   string = "java"
)

var SupportedRuntimes []string = []string{
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

	Runtime Runtime
}

// Runtime interface must be extended for every new runtime.
// Typically, a runtime struct contains fileds that are expected in
// meta/run.yaml and implements the functions required by this interface.
type Runtime interface {
	// Validate values that were read from yaml.
	Validate() error

	// GetRunConfig produces RunConfig from your yaml values.
	GetRunConfig() (*RunConfig, error)

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
}

// PickRuntime maps runtime name into runtime struct.
func PickRuntime(runtimeName string) (Runtime, error) {
	switch runtimeName {
	case Native:
		return &nativeRuntimeSettings{}, nil
	case NodeJS:
		return &nodeJsRuntimeSettings{}, nil
	case Java:
		return &javaRuntimeSettings{}, nil
	}

	return nil, fmt.Errorf("Unknown runtime: '%s'\n", runtimeName)
}

// IsRuntimeKnown checks whether meta/run.yaml was used or not.
// When meta/run.yaml is used, runConf.Runtime object is set.
// There are three possibilities:
// -------------------------
// runConf | runConf.Runtime
// -------------------------
// nil     | nil             => runConf not not specified
// <obj>   | nil             => runConf generated based on command-line arguments (e.g. --exec, --mem)
// <obj>   | <obj>           => runConf parsed from meta/run.yaml (runtime mechanism was employed)
func IsRuntimeKnown(runConf *RunConfig) bool {
	return runConf != nil && runConf.Runtime != nil
}
