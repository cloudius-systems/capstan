package util

import (
	. "gopkg.in/check.v1"
	"time"
	"math/rand"
)

type suite struct {
	repo *Repo
}

func (s *suite) SetUpTest(c *C) {
	s.repo = NewRepo(DefaultRepositoryUrl)
	s.repo.Path = c.MkDir()
	s.repo.UseS3 = false
}

var _ = Suite(&suite{})

func (s *suite) TestGithubPackageInfoRemote(c *C) {
	s.repo.ReleaseTag = "v0.53.0"
	packageName := "osv.httpserver-api"
	appPackage := s.repo.PackageInfoRemote(packageName)
	//TODO: For now let us use sleep to prevent github REST API calls fail
	// due to rate limiting. Eventually we should mock the REST api
	// (please see https://medium.com/@tech_phil/how-to-stub-external-services-in-go-8885704e8c53)
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	c.Assert(appPackage, NotNil)
	c.Check(appPackage.Name, Equals, packageName)
}

func (s *suite) TestGithubDownloadLoaderImage(c *C) {
	s.repo.ReleaseTag = "v0.51.0"
	loaderName, err := s.repo.DownloadLoaderImage("qemu")
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	c.Assert(err, IsNil)
	c.Check(loaderName, Equals, "osv-loader")
}

func (s *suite) TestGithubListPackagesRemote(c *C) {
	s.repo.ReleaseTag = "any"
	err := s.repo.ListPackagesRemote("")
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	c.Assert(err, IsNil)
}

func (s *suite) TestGithubDownloadPackageRemote(c *C) {
	s.repo.ReleaseTag = "v0.53.0"
	err := s.repo.DownloadPackageRemote("osv.httpserver-api")
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	c.Assert(err, IsNil)
}
