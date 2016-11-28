package core

import (
	"fmt"
	"github.com/cloudius-systems/capstan/runtime"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type basicRunSettings struct {
	Runtime          string                            `yaml:"runtime"`
	ConfigSet        map[string]map[string]interface{} `yaml:"config_set"`
	ConfigSetDefault string                            `yaml:"config_set_default"`
}

// ParsePackageRunManifest parses meta/run.yaml file into RunConfig
func ParsePackageRunManifest(runManifestFile string, selectedConfig string) (*runtime.RunConfig, error) {

	// Take meta/run.yaml from the current directory if not provided.
	if runManifestFile == "." {
		runManifestFile = filepath.Join(runManifestFile, "meta", "run.yaml")
	}

	// Abort silently if run.yaml does not exist (since it is not required to have one)
	if _, err := os.Stat(runManifestFile); os.IsNotExist(err) {
		return nil, nil
	}

	// From here on no error is suppressed since we do not tolerate corrupted run.yaml.

	// Open file.
	data, err := ioutil.ReadFile(runManifestFile)
	if err != nil {
		return nil, err
	}

	// Parse basic fields.
	requiredConf := basicRunSettings{}
	if err := yaml.Unmarshal(data, &requiredConf); err != nil {
		return nil, err
	}

	// Pick runtime.
	theRuntime, err := runtime.PickRuntime(requiredConf.Runtime)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Resolved runtime into: %s\n", theRuntime.GetRuntimeName())

	// Override with command-line arguments.
	if selectedConfig != "" {
		requiredConf.ConfigSetDefault = selectedConfig
	}

	// Resolve named configuration.
	if requiredConf.ConfigSet != nil {
		if requiredConf.ConfigSetDefault == "" {
			return nil, fmt.Errorf("Could not resolve named configuration - configuration name not provided")
		}
		if requiredConf.ConfigSet[requiredConf.ConfigSetDefault] == nil {
			keys := ""
			for k := range requiredConf.ConfigSet {
				keys += k + ", "
			}
			keys = strings.Trim(keys, ", ")

			return nil, fmt.Errorf("Could not resolve named configuration - config_set '%s' not one of [%s]",
				requiredConf.ConfigSetDefault, keys)
		}

		fmt.Printf("Using named configuration: '%s'\n", requiredConf.ConfigSetDefault)

		// Replace whole data with appropriate subsection of yaml only.
		data, _ = yaml.Marshal(requiredConf.ConfigSet[requiredConf.ConfigSetDefault])
	} else {
		fmt.Println("Single-configuration mode detected")
	}

	// Parse runtime-specific settings.
	yaml.Unmarshal(data, theRuntime)

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
