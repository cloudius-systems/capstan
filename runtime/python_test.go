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
	"gopkg.in/yaml.v2"
)

type pythonSuite struct {
}

var _ = Suite(&pythonSuite{})

func (*pythonSuite) TestGetBootCmd(c *C) {
	// Simulate node-4.4.5's meta/run.yaml being parsed.
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
