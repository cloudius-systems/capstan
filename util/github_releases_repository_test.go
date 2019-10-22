package util

import (
	"github.com/cloudius-systems/capstan/testing"
	. "gopkg.in/check.v1"
	"net/http/httptest"
)

func (s *suite) SetUpSuite(c *C) {
	s.server = testing.MockGitHubApiServer()
}

func (s *suite) TearDownSuite(c *C) {
	s.server.Close()
}

type suite struct {
	repo   *Repo
	server *httptest.Server
}

func (s *suite) SetUpTest(c *C) {
	s.repo = NewRepo(DefaultRepositoryUrl)
	s.repo.Path = c.MkDir()
	s.repo.UseS3 = false
	s.repo.GithubURL = s.server.URL
}

var _ = Suite(&suite{})

func (s *suite) TestGithubPackageInfoRemote(c *C) {
	s.repo.ReleaseTag = "v0.53.0"
	packageName := "osv.httpserver-api"
	appPackage := s.repo.PackageInfoRemote(packageName)
	c.Assert(appPackage, NotNil)
	c.Check(appPackage.Name, Equals, packageName)
}

func (s *suite) TestGithubDownloadLoaderImage(c *C) {
	s.repo.ReleaseTag = "v0.51.0"
	loaderName, err := s.repo.DownloadLoaderImage("qemu")
	c.Assert(err, IsNil)
	c.Check(loaderName, Equals, "osv-loader")
}

func (s *suite) TestGithubListPackagesRemote(c *C) {
	s.repo.ReleaseTag = "any"
	err := s.repo.ListPackagesRemote("")
	c.Assert(err, IsNil)
}

func (s *suite) TestGithubDownloadPackageRemote(c *C) {
	s.repo.ReleaseTag = "v0.53.0"
	err := s.repo.DownloadPackageRemote("osv.httpserver-api")
	c.Assert(err, IsNil)
}
