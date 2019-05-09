/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime

import (
	"fmt"
	"strings"

	"github.com/cloudius-systems/capstan/util"
)

// javaPackages specifies what packages are fully compatible with this runtime.
// For the time being, these are:
//   openjdk8-zulu-compact1
//   openjdk8-zulu-compact3-with-java-beans
//   openjdk7
var javaPackages = []string{"^openjdk.*"}

type javaRuntime struct {
	CommonRuntime `yaml:"-,inline"`
	Xms           string   `yaml:"xms"`
	Xmx           string   `yaml:"xmx"`
	Classpath     []string `yaml:"classpath"`
	JvmArgs       []string `yaml:"jvm_args"`
	Main          string   `yaml:"main"`
	Args          []string `yaml:"args"`
}

//
// Interface implementation
//

func (conf javaRuntime) GetRuntimeName() string {
	return string(Java)
}
func (conf javaRuntime) GetRuntimeDescription() string {
	return "Run Java application"
}
func (conf javaRuntime) GetDependencies() []string {
	return []string{"openjdk8-zulu-compact1"}
}
func (conf javaRuntime) Validate() error {
	// Only validate java-specific environment variables when base is openjdk-like.
	if isCompatibleBase(conf.Base, javaPackages) {
		if conf.Main == "" {
			return fmt.Errorf("'main' must be provided")
		}
	} else {
		if conf.Xms != "" || conf.Xmx != "" || len(conf.Classpath) > 0 ||
			len(conf.JvmArgs) > 0 || conf.Main != "" || len(conf.Args) > 0 {
			return fmt.Errorf("incompatible arguments specified [xms,xmx,classpath,jvm_args,main,args] for custom 'base'")
		}
	}

	return conf.CommonRuntime.Validate()
}
func (conf javaRuntime) GetBootCmd(cmdConfs map[string]*CmdConfig, env map[string]string) (string, error) {
	if conf.Base == "" { // Allow user to use e.g. "openjdk7:java" package instead default one.
		conf.Base = "openjdk8-zulu-compact1:java"
	}

	// Only set java-specific environment variables when base is openjdk-like.
	if isCompatibleBase(conf.Base, javaPackages) {
		if len(conf.Classpath) == 0 {
			conf.Classpath = append(conf.Classpath, "/")
		}
		if strings.HasSuffix(conf.Main, ".jar") && !util.StringInSlice("-jar", conf.JvmArgs) {
			conf.JvmArgs = append(conf.JvmArgs, "-jar")
		}
		conf.setDefaultEnv(map[string]string{
			"XMS":       conf.Xms,
			"XMX":       conf.Xmx,
			"CLASSPATH": strings.Join(conf.Classpath, ":"),
			"JVM_ARGS":  conf.concatJvmArgs(),
			"MAIN":      conf.Main,
			"ARGS":      strings.Join(conf.Args, " "),
		})
	}

	return conf.CommonRuntime.BuildBootCmd("", cmdConfs, env)
}
func (conf javaRuntime) GetYamlTemplate() string {
	return `
# REQUIRED
# Fully classified name of the main class.
# Example value: main.Hello
main: <name>

# OPTIONAL
# A list of paths where classes and other resources can be found.
# By default, the unikernel root "/" is added to the classpath.
# Example value: classpath:
#                   - /
#                   - /src
classpath:
   - <list>

# OPTIONAL
# Initial and maximum JVM memory size.
# Example value: xms: 512m
xms: <value>
xmx: <value>

# OPTIONAL
# A list of JVM args.
# Example value: jvm_args:
#                   - -Djava.net.preferIPv4Stack=true
#                   - -Dhadoop.log.dir=/hdfs/logs
jvm_args:
   - <list>

# OPTIONAL
# A list of command line args used by the application.
# Example value: args:
#                   - argument1
#                   - argument2
args:
   - <list>
` + conf.CommonRuntime.GetYamlTemplate()
}

//
// Utility
//

func (conf javaRuntime) concatJvmArgs() string {
	if len(conf.JvmArgs) > 0 {
		return strings.Join(conf.JvmArgs, " ")
	} else {
		// This is a workaround since runscript is currently unable to
		// handle empty environment variable as a parameter. So we set
		// dummy value unless user provided some actual value.
		return "-Dx=y"
	}
}
