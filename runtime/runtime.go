/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/util"
)

type RuntimeType string

const (
	Native RuntimeType = "native"
	NodeJS RuntimeType = "node"
	Java   RuntimeType = "java"
	Python RuntimeType = "python"
)

var SupportedRuntimes []RuntimeType = []RuntimeType{
	Native,
	NodeJS,
	Java,
	Python,
}

type RunConfig struct {
	InstanceName string // general
	Verbose      bool
	GCEUploadDir string
	Cmd          string
	Persist      bool
	Hypervisor   string
	ImageName    string // storage
	Volumes      []string
	Memory       string // resources
	Cpus         int
	Networking   string // networking
	Bridge       string
	NatRules     []nat.Rule
	MAC          string
}

// Runtime interface must be extended for every new runtime.
// Typically, a runtime struct contains fileds that are expected in
// meta/run.yaml and implements the functions required by this interface.
type Runtime interface {
	// Validate values that were read from yaml.
	Validate() error

	// GetBootCmd produces bootcmd based on meta/run.yaml. The cmdConfs
	// argument contains CmdConfig objects for all required packages and
	// can be used when building boot command.
	GetBootCmd(cmdConfs map[string]*CmdConfig, env map[string]string) (string, error)

	// GetRuntimeName returns unique runtime name
	// (use constant from the SupportedRuntimes list)
	GetRuntimeName() string

	// GetRuntimeDescription provides short description about what
	// is this runtime used for, 50 chars
	GetRuntimeDescription() string

	// GetYamlTemplate provides a string containing yaml content with
	// as much help text as possible.
	// NOTE: provide only runtime-specific part of yaml, see runtime/node.go for example.
	// NOTE: Write each comment in its own line for --plain flag to remove it.
	GetYamlTemplate() string

	// GetDependencies returns a list of dependent package names.
	GetDependencies() []string
}

// CommonRuntime fields are those common to all runtimes.
// This fields are set for each named-configuration separately, nothing
// is shared.
type CommonRuntime struct {
	Env  map[string]string `yaml:"env"`
	Base string            `yaml:"base"`
}

// setDefaultEnv sets default values for runtime's own environment.
// Only non-existing keys are set and then a list of those is returned.
func (r *CommonRuntime) setDefaultEnv(env map[string]string) []string {
	if r.Env == nil {
		r.Env = make(map[string]string, 15)
	}

	updated := make([]string, 15)
	for key, value := range env {
		if value == "" {
			continue
		}
		if _, exists := r.Env[key]; !exists {
			r.Env[key] = value
			updated = append(updated, key)
		}
	}
	return updated
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

# OPTIONAL
# Configuration to contextualize.
base: "<package-name>:<config_set>"
`
}

func (r CommonRuntime) Validate() error {
	for k, v := range r.Env {
		if strings.Contains(k, " ") || strings.Contains(v, " ") {
			return fmt.Errorf("spaces not allowed in env key/value: '%s':'%s'", k, v)
		}
	}

	// Common validation in case of bootcmd inheritance
	if r.Base != "" {
		if !strings.Contains(r.Base, ":") {
			return fmt.Errorf("'base' must be in format <pkg_name>:<config_set>")
		}
	}

	return nil
}

// BuildBootCmd equips runtime-specific bootcmd with common parts.
func (r CommonRuntime) BuildBootCmd(bootCmd string, cmdConfs map[string]*CmdConfig, env map[string]string) (string, error) {
	util.ExtendMap(env, r.Env)

	if r.Base != "" {
		return r.inheritBootCmd(cmdConfs, env)
	}

	// Prepend environment variables
	newBootCmd, err := PrependEnvsPrefix(bootCmd, env, true)
	if err != nil {
		return "", err
	}

	return newBootCmd, nil
}

// inheritBootCmd builds boot command based on the package referenced by "base".
func (r CommonRuntime) inheritBootCmd(cmdConfs map[string]*CmdConfig, env map[string]string) (string, error) {
	pkgName, configSet := parseBase(r.Base)

	if _, exists := cmdConfs[pkgName]; !exists || cmdConfs[pkgName] == nil {
		return "", fmt.Errorf("Failed to inherit from '%s': package not included or has no meta/run.yaml", pkgName)
	}
	if _, exists := cmdConfs[pkgName].ConfigSets[configSet]; !exists {
		return "", fmt.Errorf("Failed to inherit '%s:%s': config_set does not exist", pkgName, configSet)
	}

	original := cmdConfs[pkgName].ConfigSets[configSet]
	bootCmd, err := original.GetBootCmd(cmdConfs, env)
	if err != nil {
		return "", err
	}

	return bootCmd, nil
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
	case Python:
		return &pythonRuntime{}, nil
	}

	return nil, fmt.Errorf("Unknown runtime: '%s'\n", runtimeName)
}

// PrependEnvsPrefix prepends all key-values of env map to the boot cmd give.
// It prepends each pair in a form of "--env={KEY}={VALUE}".
// Also performs check that neither key nor value contains space.
// Argument `soft` means that operator '?=' is used that only sets env
// variable if it's not set yet.
func PrependEnvsPrefix(cmd string, env map[string]string, soft bool) (string, error) {
	operator := "="
	if soft {
		operator = "?="
	}

	s := ""
	for k, v := range env {
		s += fmt.Sprintf("--env=%s%s%s ", k, operator, v)
	}
	return fmt.Sprintf("%s%s", s, cmd), nil
}

// BootCmdForScript returns boot command that is to be used
// to run config set with name bootName.
func BootCmdForScript(bootNames []string) string {
	if len(bootNames) == 0 {
		return ""
	}
	bootCmd := ""
	for _, bootName := range bootNames {
		bootCmd += fmt.Sprintf("runscript /run/%s;", strings.TrimSpace(bootName))
	}

	return bootCmd
}

// parseBase parses base into pkgName and configSetName.
// We assume no error can occur, so validation needs to be performed.
func parseBase(base string) (string, string) {
	parts := strings.SplitN(base, ":", 2)
	return parts[0], parts[1]
}

// isCompatibleBase tells whether base provided is compatible.
// The purpose of "compatiblity check" is that we differ between two
// types of bases: those that behave exactly like the runtime's
// default base. E.g. for Java runtime, compatible bases are
// 'openjdk7', 'openjdk8-zulu-compact1', ... while incompatible
// bases would be e.g. 'osv.cli' or 'apache.spark-2.1.1'. Point is
// that we set runtime-specific environment variables (e.g. JVM_ARGS)
// only for compatible bases and not for the incompatible ones.
func isCompatibleBase(base string, patterns []string) bool {
	if base == "" {
		return true
	}

	pkgName, _ := parseBase(base)

	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(pkgName) {
			return true
		}
	}
	return false
}
