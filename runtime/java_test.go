/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime

import (
	"testing"

	. "github.com/cloudius-systems/capstan/testing"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v2"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type javaSuite struct {
}

var _ = Suite(&javaSuite{})

func (*javaSuite) TestGetBootCmd(c *C) {
	// Simulate openjdk8-zulu-compact1's meta/run.yaml being parsed.
	cmdConfs := map[string]*CmdConfig{
		"openjdk8-zulu-compact1": &CmdConfig{
			RuntimeType:      Native,
			ConfigSetDefault: "java",
			ConfigSets: map[string]Runtime{
				"java": nativeRuntime{
					BootCmd: "/java.so",
					CommonRuntime: CommonRuntime{
						Env: map[string]string{
							"XMS":       "512m",
							"XMX":       "512m",
							"CLASSPATH": "/",
							"JVM_ARGS":  "-Duser.dir=/",
							"MAIN":      "main.Hello",
							"ARGS":      "",
						},
					},
				},
			},
		},
		"openjdk7": &CmdConfig{
			RuntimeType:      Native,
			ConfigSetDefault: "java",
			ConfigSets: map[string]Runtime{
				"java": nativeRuntime{
					BootCmd: "/java7.so",
					CommonRuntime: CommonRuntime{
						Env: map[string]string{
							"XMS":       "512m",
							"XMX":       "512m",
							"CLASSPATH": "/",
							"JVM_ARGS":  "-Duser.dir=/",
							"MAIN":      "main.Hello",
							"ARGS":      "",
						},
					},
				},
			},
		},
		"mypackage": &CmdConfig{
			RuntimeType:      Native,
			ConfigSetDefault: "run",
			ConfigSets: map[string]Runtime{
				"run": nativeRuntime{
					BootCmd: "/mypackage.so",
				},
			},
		},
	}

	m := []struct {
		comment      string
		runYamlText  string
		expectedBoot string
		expectedEnv  []string
	}{
		{
			"simple",
			`
			runtime: java
			config_set:
			  default:
			    main: demo.Main
			    classpath:
			      - /src
			`,
			"/java.so", []string{
				"--env=XMS?=512m",
				"--env=XMX?=512m",
				"--env=CLASSPATH?=/src",
				"--env=JVM_ARGS?=-Dx=y",
				"--env=MAIN?=demo.Main",
				"--env=ARGS?=",
			},
		},
		{
			"multiple classpath",
			`
			runtime: java
			config_set:
			  default:
			    main: demo.Main
			    classpath:
			      - /src
			      - /src/package1
			`,
			"/java.so", []string{
				"--env=XMS?=512m",
				"--env=XMX?=512m",
				"--env=CLASSPATH?=/src:/src/package1",
				"--env=JVM_ARGS?=-Dx=y",
				"--env=MAIN?=demo.Main",
				"--env=ARGS?=",
			},
		},
		{
			"xms and xmx",
			`
			runtime: java
			config_set:
			  default:
			    main: demo.Main
			    classpath:
			      - /src
			    xms: 128m
			    xmx: 1024m
			`,
			"/java.so", []string{
				"--env=XMS?=128m",
				"--env=XMX?=1024m",
				"--env=CLASSPATH?=/src",
				"--env=JVM_ARGS?=-Dx=y",
				"--env=MAIN?=demo.Main",
				"--env=ARGS?=",
			},
		},
		{
			"jvm args",
			`
			runtime: java
			config_set:
			  default:
			    main: demo.Main
			    classpath:
			      - /src
			    jvm_args:
			      - -Darg1=val1
			      - -Darg2=val2
			`,
			"/java.so", []string{
				"--env=XMS?=512m",
				"--env=XMX?=512m",
				"--env=CLASSPATH?=/src",
				"--env=JVM_ARGS?=-Darg1=val1 -Darg2=val2",
				"--env=MAIN?=demo.Main",
				"--env=ARGS?=",
			},
		},
		{
			"args",
			`
			runtime: java
			config_set:
			  default:
			    main: demo.Main
			    classpath:
			      - /src
			    args:
			      - localhost
			      - 8000
			`,
			"/java.so", []string{
				"--env=XMS?=512m",
				"--env=XMX?=512m",
				"--env=CLASSPATH?=/src",
				"--env=JVM_ARGS?=-Dx=y",
				"--env=MAIN?=demo.Main",
				"--env=ARGS?=localhost 8000",
			},
		},
		{
			"jar as main",
			`
			runtime: java
			config_set:
			  default:
			    main: app.jar
			`,
			"/java.so", []string{
				"--env=XMS?=512m",
				"--env=XMX?=512m",
				"--env=CLASSPATH?=/",
				"--env=JVM_ARGS?=-jar",
				"--env=MAIN?=app.jar",
				"--env=ARGS?=",
			},
		},
		{
			"jar as main and -jar already provided",
			`
			runtime: java
			config_set:
			  default:
			    main: app.jar
			    jvm_args:
			      - "-jar"
			`,
			"/java.so", []string{
				"--env=XMS?=512m",
				"--env=XMX?=512m",
				"--env=CLASSPATH?=/",
				"--env=JVM_ARGS?=-jar",
				"--env=MAIN?=app.jar",
				"--env=ARGS?=",
			},
		},
		{
			"jar as main and -jar already provided among other args",
			`
			runtime: java
			config_set:
			  default:
			    main: app.jar
			    jvm_args:
			      - "-jar"
			      - "-xyz"
			`,
			"/java.so", []string{
				"--env=XMS?=512m",
				"--env=XMX?=512m",
				"--env=CLASSPATH?=/",
				"--env=JVM_ARGS?=-jar -xyz",
				"--env=MAIN?=app.jar",
				"--env=ARGS?=",
			},
		},
		{
			"different openjdk package",
			`
			runtime: java
			config_set:
			  default:
			    base: "openjdk7:java"
			    main: demo.Main
			    classpath:
			      - /src
			`,
			"/java7.so", []string{
				"--env=XMS?=512m",
				"--env=XMX?=512m",
				"--env=CLASSPATH?=/src",
				"--env=JVM_ARGS?=-Dx=y",
				"--env=MAIN?=demo.Main",
				"--env=ARGS?=",
			},
		},
		{
			"different base package",
			`
			runtime: java
			config_set:
			  default:
			    base: "mypackage:run"
			`,
			"/mypackage.so", []string{},
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare
		cmdConfig, err := ParsePackageRunManifestData([]byte(FixIndent(args.runYamlText)))
		testRuntime, _ := cmdConfig.selectConfigSetByName("default")

		// This is what we're testing here.
		boot, err := testRuntime.GetBootCmd(cmdConfs, map[string]string{})

		// Expectations.
		c.Assert(err, IsNil)
		c.Check(boot, BootCmdEquals, args.expectedBoot, args.expectedEnv)
	}
}

func (*javaSuite) TestValidate(c *C) {
	m := []struct {
		comment     string
		runYamlText string
		err         string
	}{
		{
			"missing main",
			`
			runtime: java
			config_set:
			  default:
			    classpath:
			      - /src
			`,
			"'main' must be provided",
		},
		{
			"inheritance overrides validation",
			`
			runtime: java
			config_set:
			  default:
			    base: "some.package:its_config_set"
			`,
			"",
		},
		{
			"incompatible with 'base' - xms",
			`
			runtime: java
			config_set:
			  default:
			    base: "foo:bar"
			    xms: foo.bar
			`,
			"incompatible arguments specified \\[xms,xmx,classpath,jvm_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - xmx",
			`
			runtime: java
			config_set:
			  default:
			    base: "foo:bar"
			    xmx: foo.bar
			`,
			"incompatible arguments specified \\[xms,xmx,classpath,jvm_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - classpath",
			`
			runtime: java
			config_set:
			  default:
			    base: "foo:bar"
			    classpath:
			      - foo.bar
			`,
			"incompatible arguments specified \\[xms,xmx,classpath,jvm_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - jvm_args",
			`
			runtime: java
			config_set:
			  default:
			    base: "foo:bar"
			    jvm_args:
			      - foo.bar
			`,
			"incompatible arguments specified \\[xms,xmx,classpath,jvm_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - main",
			`
			runtime: java
			config_set:
			  default:
			    base: "foo:bar"
			    main: foo.bar
			`,
			"incompatible arguments specified \\[xms,xmx,classpath,jvm_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - args",
			`
			runtime: java
			config_set:
			  default:
			    base: "foo:bar"
			    args:
			      - foo.bar
			`,
			"incompatible arguments specified \\[xms,xmx,classpath,jvm_args,main,args\\] for custom 'base'",
		},
		{
			"compatible package behaves same as defult one",
			`
			runtime: java
			config_set:
			  default:
			    base: "openjdk7:java"
			`,
			"'main' must be provided",
		},
	}
	for i, args := range m {
		c.Logf("CASE #%d: %s", i, args.comment)

		// Prepare
		cmdConfig, err := ParsePackageRunManifestData([]byte(FixIndent(args.runYamlText)))
		testRuntime, _ := cmdConfig.selectConfigSetByName("default")

		// This is what we're testing here.
		err = testRuntime.Validate()

		// Expectations.
		if args.err == "" {
			c.Check(err, IsNil)
		} else {
			c.Check(err, ErrorMatches, args.err)
		}
	}
}

func (*javaSuite) TestGetYamlTemplateIsComplete(c *C) {
	// Prepare
	testRuntime := javaRuntime{}

	// This is what we're testing here.
	template := testRuntime.GetYamlTemplate()

	// Expectations.
	c.Check(template, MatchesMultiline, "xms:")
	c.Check(template, MatchesMultiline, "xmx:")
	c.Check(template, MatchesMultiline, "classpath:")
	c.Check(template, MatchesMultiline, "jvm_args:")
	c.Check(template, MatchesMultiline, "main:")
	c.Check(template, MatchesMultiline, "args:")
	c.Check(template, MatchesMultiline, "env:")
}

func (*javaSuite) TestGetYamlTemplateIsValidYaml(c *C) {
	// Prepare
	testRuntime := javaRuntime{}
	template := testRuntime.GetYamlTemplate()

	// This is what we're testing here.
	err := yaml.Unmarshal([]byte(template), &javaRuntime{})

	// Expectations.
	c.Assert(err, IsNil)
}
