package core

import (
	"fmt"
	"github.com/mikelangelo-project/capstan/runtime"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// cmdConfigInternal is used just for meta/run.yaml unmarshalling.
type cmdConfigInternal struct {
	Runtime          runtime.RuntimeType               `yaml:"runtime"`
	ConfigSet        map[string]map[string]interface{} `yaml:"config_set"`
	ConfigSetDefault string                            `yaml:"config_set_default"`
}

// CmdConfig is a result that parsing meta/run.yaml yields.
type CmdConfig struct {
	RuntimeType      runtime.RuntimeType
	ConfigSetDefault string

	// ConfigSets is a map of available <config-name>:<runtime> pairs.
	// The map is built based on meta/run.yaml.
	ConfigSets map[string]runtime.Runtime
}

// ParsePackageRunManifest parses meta/run.yaml file into RunConfig.
func ParsePackageRunManifest(cmdConfigFile string, selectedConfig string) (*runtime.RunConfig, error) {

	// Take meta/run.yaml from the current directory if not provided.
	if cmdConfigFile == "." {
		cmdConfigFile = filepath.Join(cmdConfigFile, "meta", "run.yaml")
	}

	// Abort silently if run.yaml does not exist (since it is not required to have one)
	if _, err := os.Stat(cmdConfigFile); os.IsNotExist(err) {
		return nil, nil
	}

	// From here on, no error is suppressed since we do not tolerate corrupted run.yaml.

	// Open file.
	data, err := ioutil.ReadFile(cmdConfigFile)
	if err != nil {
		return nil, err
	}

	// Parse.
	runManif, err := ParsePackageRunManifestData(data)
	if err != nil {
		return nil, err
	}

	// At this point we are certain that runtime name is valid, so we
	// confirm this to user.
	fmt.Printf("Resolved runtime into: %s\n", runManif.RuntimeType)

	// Override with command-line argument.
	if selectedConfig != "" {
		runManif.ConfigSetDefault = selectedConfig
	}

	// Select one.
	theRuntime, err := runManif.selectConfigSetByName(runManif.ConfigSetDefault)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Using named configuration: '%s'\n", runManif.ConfigSetDefault)

	// Validate.
	if err := theRuntime.Validate(); err != nil {
		return nil, fmt.Errorf("Runtime validation failed: %s\n", err)
	}

	// Convert runtime config to common run config.
	conf, err := theRuntime.GetRunConfig()
	if err != nil {
		return nil, err
	}

	// Prepend environment variables
	if conf.Cmd, err = runtime.PrependEnvsPrefix(conf.Cmd, theRuntime.GetEnv()); err != nil {
		return nil, err
	}

	// Remember original config as well.
	conf.Runtime = theRuntime

	return conf, nil
}

// ParsePackageRunManifestData returns parsed manifest data.
func ParsePackageRunManifestData(cmdConfigData []byte) (*CmdConfig, error) {
	res := CmdConfig{}

	// Parse basic fields.
	internal := cmdConfigInternal{}
	if err := yaml.Unmarshal(cmdConfigData, &internal); err != nil {
		return nil, fmt.Errorf("failed to parse meta/run.yaml: %s", err)
	} else {
		// Store basic fields into result struct
		res.RuntimeType = internal.Runtime
		res.ConfigSetDefault = internal.ConfigSetDefault
	}

	res.ConfigSets = make(map[string]runtime.Runtime)

	// We are marshalling the `map[interface{}]interface{}` data here (containing single
	// configuration set parameters) so that we will be able to unmarshal it in the next
	// step into the appropriate structure. This trick is used so that we do not need to
	// trouble with casting interfaces to extract config set parameters - we leave it to
	// yaml unmarshaller instead. In other words, we parse meta/run.yaml per partes:
	// config_set:
	//    name1: <map[interface{}]interface{}>  # <--- 1st part
	//    name2: <map[interface{}]interface{}>  # <--- 2nd part
	//    name3: <map[interface{}]interface{}>  # <--- 3rd part
	// Each part is unmarshalled into one interface.
	// Variable 'subdata' in the following for loop contains yaml string representing
	// single configuration set data that we then unmarshall into appropriate runtime
	// interface.
	for k := range internal.ConfigSet {
		// Prepare empty runtime struct that will be used for unmarshalling.
		theRuntime, err := runtime.PickRuntime(internal.Runtime)
		if err != nil {
			return nil, err
		}

		// Use appropriate subsection of yaml only.
		subdata, _ := yaml.Marshal(internal.ConfigSet[k])

		// Parse runtime-specific settings.
		if err := yaml.Unmarshal(subdata, theRuntime); err != nil {
			return nil, fmt.Errorf("failed to parse data for configset '%s': %s", k, err)
		}

		res.ConfigSets[k] = theRuntime
	}

	if len(res.ConfigSets) == 0 {
		return nil, fmt.Errorf("failed to parse meta/run.yaml: at least one config_set must be provided")
	}

	return &res, nil
}

// selectConfigSetByName selects appropriate config set and returns it.
func (r *CmdConfig) selectConfigSetByName(name string) (runtime.Runtime, error) {
	availableNames := fmt.Sprintf("['%s']", strings.Join(keysOfMap(r.ConfigSets), "', '"))

	// Handle unspecified configuration name.
	if name == "" && len(r.ConfigSets) == 1 {
		// If only one configuration set is provided, then there is no doubt.
		for k := range r.ConfigSets {
			return r.ConfigSets[k], nil
		}
	} else if name == "" {
		return nil, fmt.Errorf("Could not select which configuration set to run:\n"+
			"Neither --runconfig <name> is provided, nor config_set_default is set in meta/run.yaml\n"+
			"Available names: %s", availableNames)
	}

	if r.ConfigSets[name] == nil {
		return nil, fmt.Errorf("Could not select which configuration set to run:\n"+
			"Configuration set name '%s' not one of %s",
			name, availableNames)
	}

	return r.ConfigSets[name], nil
}

// keysOfMap does nothing but returns a list of all the keys in a map.
func keysOfMap(myMap map[string]runtime.Runtime) []string {
	keys := make([]string, len(myMap))
	i := 0
	for k := range myMap {
		keys[i] = k
		i++
	}
	return keys
}
