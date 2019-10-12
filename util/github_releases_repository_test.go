package util

import (
	"fmt"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
)

func (s *suite) SetUpSuite(c *C) {
	s.server = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := "testdata/github" + r.RequestURI + "/payload"
			fmt.Printf("httptest: Mocking: %s with %s \n", r.RequestURI, path)
			if payload, err := ioutil.ReadFile(path); err == nil {
				payloadStr := string(payload)
				payloadStr = strings.ReplaceAll(payloadStr, "https://github.com", s.server.URL)
				w.Write([]byte(payloadStr))
			} else {
				http.Error(w, "not found", http.StatusNotFound)
			}
		}))
}

func (s *suite) TearDownSuite(c *C) {
	s.server.Close()
}

type suite struct {
	repo *Repo
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
