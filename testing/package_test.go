package testing

import (
	"github.com/cloudius-systems/capstan/core"
	. "gopkg.in/check.v1"
)

type TestingPackageSuite struct {
}

var _ = Suite(&TestingPackageSuite{})

func (s *TestingPackageSuite) TestPackageUnmarshaller(c *C) {
	packageYaml := "name: Capstan tester\nauthor: MIKE\nversion: 0.23-24-gc60331d\n" +
		"require:\n - httpserver\n - openmpi\n" +
		"binary:\n app: /usr/bin/app.so\n /usr/bin/app: /usr/local/bin/app.so"

	var simplePackage core.Package
	err := simplePackage.Parse([]byte(packageYaml))

	c.Assert(err, IsNil)
	c.Assert(simplePackage.Name, Equals, "Capstan tester")
	c.Assert(simplePackage.Author, Equals, "MIKE")
	c.Assert(simplePackage.Version, Equals, "0.23-24-gc60331d")
	c.Assert(simplePackage.Require, HasLen, 2)
	c.Assert(simplePackage.Binary["app"], Equals, "/usr/bin/app.so")
	c.Assert(simplePackage.Binary["/usr/bin/app"], Equals, "/usr/local/bin/app.so")
}

func (s *TestingPackageSuite) TestInvalidYaml(c *C) {
	packageYaml := "name Capstan tester"

	var pkg core.Package
	err := pkg.Parse([]byte(packageYaml))

	c.Assert(err, NotNil)
}

func (s *TestingPackageSuite) TestIncomplete(c *C) {
	emptyYaml := ""

	var emptyPackage core.Package
	err := emptyPackage.Parse([]byte(emptyYaml))

	c.Assert(err, NotNil)

	nameYaml := "name: MIKE test"
	var namePackage core.Package
	err = namePackage.Parse([]byte(nameYaml))
	c.Assert(err, NotNil)
}

func (s *TestingPackageSuite) TestMinimalPackageYaml(c *C) {
	minimalYaml := "name: MIKE test\nauthor: MIKE"
	var nameAuthorPackage core.Package
	err := nameAuthorPackage.Parse([]byte(minimalYaml))
	c.Assert(err, IsNil)
}
