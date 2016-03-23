package core

import (
	. "gopkg.in/check.v1"
	"os"
	"path/filepath"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type javaSuite struct{}

var _ = Suite(&javaSuite{})

var javaParseTests = []struct {
	Yaml           string
	JavaObj        *Java
	Err            string
	CommandLineStr string
	VmArgsStr      string
}{
	{"", nil, "'main' must be provided", "", ""},
	{"main: Main", nil, "'classpath' must be provided", "", ""},
	{
		"main: Main\nclasspath:\n - /hello\n - /test",
		&Java{Main: "Main", Classpath: []string{"/hello", "/test"}},
		"",
		"-cp /hello:/test Main",
		"",
	},
	{
		"main: Main\nclasspath:\n - /hello\n - /test\nargs:\n - hello\n - world",
		&Java{Main: "Main", Classpath: []string{"/hello", "/test"}, Args: []string{"hello", "world"}},
		"",
		"-cp /hello:/test Main hello world",
		"",
	},
	{
		"main: Main\nclasspath:\n - /hello\n - /test\nargs:\n - hello\n - world\nvmargs:\n - val1\n - val2",
		&Java{Main: "Main", Classpath: []string{"/hello", "/test"}, Args: []string{"hello", "world"}, VmArgs: []string{"val1", "val2"}},
		"",
		"-cp /hello:/test Main hello world",
		"-val1 -val2",
	},
	{
		"main: Main\nclasspath:\n - /hello\n - /test\nvmargs:\n - val2\n - val1\n - Xms256MB\n - XX:+CMSClassUnloadingEnabled",
		&Java{Main: "Main", Classpath: []string{"/hello", "/test"}, Args: nil, VmArgs: []string{"val2", "val1", "Xms256MB", "XX:+CMSClassUnloadingEnabled"}},
		"",
		"-cp /hello:/test Main",
		"-val2 -val1 -Xms256MB -XX:+CMSClassUnloadingEnabled",
	},
}

var javaCommandLineTests = []struct {
	Main        string
	Classpath   []string
	Args        []string
	CommandLine string
}{
	{"Main", nil, nil, "Main"},
	{"Main", []string{}, []string{}, "Main"},
	{"Main", []string{"/cp1", "/cp2/test.jar"}, nil, "-cp /cp1:/cp2/test.jar Main"},
	{"Main", []string{"/cp1", "/cp2/test.jar"}, []string{}, "-cp /cp1:/cp2/test.jar Main"},
	{"Main", nil, []string{"hello", "test"}, "Main hello test"},
	{"Main", []string{"/cp1", "/cp2/test.jar"}, []string{"hello", "test"}, "-cp /cp1:/cp2/test.jar Main hello test"},
}

func (s *javaSuite) TestJavaParsing(c *C) {
	for _, test := range javaParseTests {
		var java Java
		err := java.Parse([]byte(test.Yaml))

		if test.Err != "" {
			c.Assert(err, ErrorMatches, test.Err)
		} else {
			// Make sure there was no error unmarshalling the Yaml
			c.Assert(err, Equals, nil)

			// Check the content of parsed struct
			c.Assert(java, DeepEquals, *test.JavaObj)

			// Check command line
			c.Assert(java.GetCommandLine(), Equals, test.CommandLineStr)

			// Also check the VM args
			c.Assert(java.GetVmArgs(), Equals, test.VmArgsStr)
		}
	}
}

func (s *javaSuite) TestCommandLines(c *C) {
	for _, test := range javaCommandLineTests {
		java := &Java{Main: test.Main, Classpath: test.Classpath, Args: test.Args}
		c.Assert(java.GetCommandLine(), Equals, test.CommandLine)
	}
}

func (s *javaSuite) TestIsJavaPackage(c *C) {
	packageDir := filepath.Join(os.TempDir(), "test-pkg")
	metaDir := filepath.Join(packageDir, "meta")
	javaConfig := filepath.Join(metaDir, "java.yaml")

	// Prepare empty package dir.
	os.MkdirAll(metaDir, 0777)
	defer os.RemoveAll(packageDir)

	// A package without java.yaml file is not Java package.
	c.Assert(IsJavaPackage(packageDir), Equals, false)

	// Try to create an empty java config file.
	if _, err := os.Create(javaConfig); err != nil {
		c.Fail()
	}

	// This should now be seen as Java package
	c.Assert(IsJavaPackage(packageDir), Equals, true)
}
