/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime

import (
	. "github.com/mikelangelo-project/capstan/testing"
	. "gopkg.in/check.v1"
	yaml "gopkg.in/yaml.v1"
)

type nodeSuite struct {
}

var _ = Suite(&nodeSuite{})

func (*nodeSuite) TestGetBootCmd(c *C) {
	// Simulate node-4.4.5's meta/run.yaml being parsed.
	cmdConfs := map[string]*CmdConfig{
		"node-4.4.5": &CmdConfig{
			RuntimeType:      Native,
			ConfigSetDefault: "node",
			ConfigSets: map[string]Runtime{
				"node": nativeRuntime{
					BootCmd: "/node.so",
					CommonRuntime: CommonRuntime{
						Env: map[string]string{
							"NODE_ARGS": "--",
							"MAIN":      "/mymain.js",
							"ARGS":      "",
						},
					},
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
			runtime: node
			config_set:
			  default:
			    main: /server.js
			`,
			"/node.so", []string{
				"--env=NODE_ARGS?=--",
				"--env=MAIN?=/server.js",
				"--env=ARGS?=",
			},
		},
		{
			"node args",
			`
			runtime: node
			config_set:
			  default:
			    main: /server.js
			    node_args:
			        - --version
			`,
			"/node.so", []string{
				"--env=NODE_ARGS?=--version",
				"--env=MAIN?=/server.js",
				"--env=ARGS?=",
			},
		},
		{
			"args",
			`
			runtime: node
			config_set:
			  default:
			    main: /server.js
			    args:
			        - localhost
			        - 8000
			`,
			"/node.so", []string{
				"--env=NODE_ARGS?=--",
				"--env=MAIN?=/server.js",
				"--env=ARGS?=localhost 8000",
			},
		},
		{
			"shell",
			`
			runtime: node
			config_set:
			  default:
			    shell: true
			`,
			"/node.so", []string{
				"--env=NODE_ARGS?=--",
				"--env=MAIN?=",
				"--env=ARGS?=",
			},
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

func (*nodeSuite) TestValidate(c *C) {
	m := []struct {
		comment     string
		runYamlText string
		err         string
	}{
		{
			"incompatible with 'base' - shell",
			`
			runtime: node
			config_set:
			  default:
			    base: "foo:bar"
			    shell: true
			`,
			"incompatible arguments specified \\[shell,node_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - node_args",
			`
			runtime: node
			config_set:
			  default:
			    base: "foo:bar"
			    node_args:
			      - foo.bar
			`,
			"incompatible arguments specified \\[shell,node_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - main",
			`
			runtime: node
			config_set:
			  default:
			    base: "foo:bar"
			    main: foo.bar
			`,
			"incompatible arguments specified \\[shell,node_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - args",
			`
			runtime: node
			config_set:
			  default:
			    base: "foo:bar"
			    args:
			      - foo.bar
			`,
			"incompatible arguments specified \\[shell,node_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'shell' - main",
			`
			runtime: node
			config_set:
			  default:
			    shell: true
			    main: /script.js
			`,
			"incompatible arguments specified \\[main,args\\] for shell=true",
		},
		{
			"incompatible with 'shell' - args",
			`
			runtime: node
			config_set:
			  default:
			    shell: true
			    args:
			      - foo.bar
			`,
			"incompatible arguments specified \\[main,args\\] for shell=true",
		},
		{
			"incompatible with 'shell' - env.MAIN",
			`
			runtime: node
			config_set:
			  default:
			    shell: true
			    env:
			      MAIN: foo
			`,
			"incompatible 'env' keys specified \\[MAIN,ARGS\\] for shell=true",
		},
		{
			"incompatible with 'shell' - env.ARGS",
			`
			runtime: node
			config_set:
			  default:
			    shell: true
			    env:
			      ARGS: foo.bar
			`,
			"incompatible 'env' keys specified \\[MAIN,ARGS\\] for shell=true",
		},
		{
			"missing main",
			`
			runtime: node
			config_set:
			  default:
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

func (*nodeSuite) TestGetYamlTemplateIsComplete(c *C) {
	// Prepare
	testRuntime := nodeJsRuntime{}

	// This is what we're testing here.
	template := testRuntime.GetYamlTemplate()

	// Expectations.
	c.Check(template, MatchesMultiline, "node_args:")
	c.Check(template, MatchesMultiline, "main:")
	c.Check(template, MatchesMultiline, "args:")
	c.Check(template, MatchesMultiline, "env:")
}

func (*nodeSuite) TestGetYamlTemplateIsValidYaml(c *C) {
	// Prepare
	testRuntime := nodeJsRuntime{}
	template := testRuntime.GetYamlTemplate()

	// This is what we're testing here.
	err := yaml.Unmarshal([]byte(template), &nodeJsRuntime{})

	// Expectations.
	c.Assert(err, IsNil)
}
