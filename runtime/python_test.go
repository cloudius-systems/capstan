/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime

import (
	. "github.com/cloudius-systems/capstan/testing"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v2"
)

type pythonSuite struct {
}

var _ = Suite(&pythonSuite{})

func (*pythonSuite) TestGetBootCmd(c *C) {
	// Simulate python-2.7's meta/run.yaml being parsed.
	cmdConfs := map[string]*CmdConfig{
		"python-2.7": &CmdConfig{
			RuntimeType:      Native,
			ConfigSetDefault: "python",
			ConfigSets: map[string]Runtime{
				"python": nativeRuntime{
					BootCmd: "/python.so",
					CommonRuntime: CommonRuntime{
						Env: map[string]string{
							"PYTHON_ARGS": "-O",
							"MAIN":        "-",
							"ARGS":        "",
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
			runtime: python
			config_set:
			  default:
			    main: /script.py
			`,
			"/python.so", []string{
				"--env=PYTHON_ARGS?=-O",
				"--env=MAIN?=/script.py",
				"--env=ARGS?=",
			},
		},
		{
			"python args",
			`
			runtime: python
			config_set:
			  default:
			    main: /script.py
			    python_args:
			        - --version
			`,
			"/python.so", []string{
				"--env=PYTHON_ARGS?=--version",
				"--env=MAIN?=/script.py",
				"--env=ARGS?=",
			},
		},
		{
			"args",
			`
			runtime: python
			config_set:
			  default:
			    main: /script.py
			    args:
			        - localhost
			        - 8000
			`,
			"/python.so", []string{
				"--env=PYTHON_ARGS?=-O",
				"--env=MAIN?=/script.py",
				"--env=ARGS?=localhost 8000",
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

func (*pythonSuite) TestValidate(c *C) {
	m := []struct {
		comment     string
		runYamlText string
		err         string
	}{
		{
			"incompatible with 'base' - shell",
			`
			runtime: python
			config_set:
			  default:
			    base: "foo:bar"
			    shell: true
			`,
			"incompatible arguments specified \\[shell,python_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - python_args",
			`
			runtime: python
			config_set:
			  default:
			    base: "foo:bar"
			    python_args:
			      - foo.bar
			`,
			"incompatible arguments specified \\[shell,python_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - main",
			`
			runtime: python
			config_set:
			  default:
			    base: "foo:bar"
			    main: foo.bar
			`,
			"incompatible arguments specified \\[shell,python_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'base' - args",
			`
			runtime: python
			config_set:
			  default:
			    base: "foo:bar"
			    args:
			      - foo.bar
			`,
			"incompatible arguments specified \\[shell,python_args,main,args\\] for custom 'base'",
		},
		{
			"incompatible with 'shell' - main",
			`
			runtime: python
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
			runtime: python
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
			runtime: python
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
			runtime: python
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
			runtime: python
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

func (*pythonSuite) TestGetYamlTemplateIsComplete(c *C) {
	// Prepare
	testRuntime := pythonRuntime{}

	// This is what we're testing here.
	template := testRuntime.GetYamlTemplate()

	// Expectations.
	c.Check(template, MatchesMultiline, "python_args:")
	c.Check(template, MatchesMultiline, "main:")
	c.Check(template, MatchesMultiline, "args:")
	c.Check(template, MatchesMultiline, "env:")
}

func (*pythonSuite) TestGetYamlTemplateIsValidYaml(c *C) {
	// Prepare
	testRuntime := pythonRuntime{}
	template := testRuntime.GetYamlTemplate()

	// This is what we're testing here.
	err := yaml.Unmarshal([]byte(template), &pythonRuntime{})

	// Expectations.
	c.Assert(err, IsNil)
}
