package util

import (
	"encoding/json"
	"fmt"
	"github.com/cloudius-systems/capstan/core"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"runtime"
)

const (
	OsvReleasesSuffix = "/repos/cloudius-systems/osv/releases"
)

type Asset struct {
	Name        string `json:"name"`
	Size        int    `json:"size"`
	DownloadUrl string `json:"browser_download_url"`
}

type Release struct {
	Id     int     `json:"id"`
	Name   string  `json:"name"`
	Tag    string  `json:"tag_name"`
	Assets []Asset `json:"assets"`
}

//GET /repos/:owner/:repo/releases/latest - get latest release with assets
// https://api.github.com/repos/cloudius-systems/osv/releases/latest

//GET /repos/:owner/:repo/releases/tags/:tag - get release by tag with assets
// https://api.github.com/repos/cloudius-systems/osv/releases/tags/v0.51.0

//GET /repos/:owner/:repo/releases - get all releases with assets
// https://api.github.com/repos/cloudius-systems/osv/releases

func (r *Repo) queryReleases() ([]Release, error) {
	apiSuffix := "/latest"
	if r.ReleaseTag == "any" {
		apiSuffix = ""
	} else if r.ReleaseTag != "latest" {
		apiSuffix = fmt.Sprintf("/tags/%s", r.ReleaseTag)
	}
	//
	// Fetch release info with assets for the latest one or identified by tag
	// or all releases each with array of assets
	responseBytes, err := r.githubMakeReleaseApiCall(apiSuffix)
	if err != nil {
		return nil, err
	}

	var releases []Release
	if r.ReleaseTag == "any" {
		if err := json.Unmarshal(responseBytes, &releases); err != nil {
			return nil, err
		}
	} else {
		var release Release
		if err := json.Unmarshal(responseBytes, &release); err != nil {
			return nil, err
		}
		releases = append(releases, release)
	}
	return releases, nil
}

func (r *Repo) githubListPackagesRemote(search string) error {
	releases, err := r.queryReleases()
	if err != nil {
		return err
	}
	fmt.Printf("%-10s%s\n", "Release", FileInfoHeader())
	for _, release := range releases {
		for _, asset := range release.Assets {
			if strings.HasPrefix(asset.Name, "osv") && strings.HasSuffix(asset.Name, ".yaml") {
				if pkg := remotePackageInfo(asset.DownloadUrl); pkg != nil && strings.Contains(pkg.Name, search) {
					fmt.Printf("%-10s%s\n", release.Tag, pkg.String())
				}
			}
		}
	}
	return nil
}

func getArch() string {
	arch := runtime.GOARCH
	if arch == "arm64" {
		return "aarch64"
	} else if arch == "amd64" {
		return "x86_64"
	} else {
		return arch
	}
}

func (r *Repo) downloadImageFromGithub(imageName string, hypervisor string, contains string) (string, error) {
	releases, err := r.queryReleases()
	if err != nil {
		return imageName, err
	}

	err = os.MkdirAll(filepath.Join(r.RepoPath(), imageName), os.ModePerm)
	if err != nil {
		return imageName, err
	}

	// Walk release by release until you find one that has file asset with correct prefix and containsFilter
	containsFilter := contains
	if containsFilter == "" {
		containsFilter = getArch()
	}
	for _, release := range releases {
		fileUrl := ""

		for _, asset := range release.Assets {
			if strings.HasPrefix(asset.Name, imageName + ".") && strings.Contains(asset.Name, containsFilter) {
				fileUrl = asset.DownloadUrl
			}

			// Both must be found for OSv loader to exist in remote repository.
			if fileUrl != "" {
				// Download the loader file itself
				fileName := fmt.Sprintf("%s/%s.%s", imageName, imageName, hypervisor)
				if err = r.downloadFile(fileUrl, r.RepoPath(), fileName); err != nil {
					return imageName, err
				}

				fmt.Printf("Downloaded image (%s) from github.\n", imageName)
				return imageName, nil
			}
		}
	}

	return imageName, fmt.Errorf(
		"The image: %s is not available in the given release (%s) in GitHub", imageName, r.ReleaseTag)
}

func (r *Repo) downloadLoaderImageFromGithub(loaderImageName string, hypervisor string) (string, error) {
	return r.downloadImageFromGithub(loaderImageName, hypervisor, hypervisor + "." + getArch())
}

// DownloadPackage downloads a package from the S3 repository into local.
func (r *Repo) downloadPackageFromGithub(packageName string) error {
	remote, err := r.getRemotePackageInfoInGithub(packageName)
	if err != nil {
		return err
	}
	// If the package is not found on a remote repository, inform the user.
	if remote == nil {
		return fmt.Errorf("package %s is not available in the given release (%s) in GitHub", packageName, r.ReleaseTag)
	}

	// Get the root of the packages dir.
	packagesRoot := r.PackagesPath()

	// Make sure the path exists by creating the entire directory structure.
	err = os.MkdirAll(packagesRoot, 0775)
	if err != nil {
		return fmt.Errorf("%s: mkdir failed", packagesRoot)
	}

	packageManifest := fmt.Sprintf("%s.yaml", packageName)
	packageFile := fmt.Sprintf("%s.mpm", packageName)

	// Download manifest file.
	err = r.downloadFile(remote.manifestURL, packagesRoot, packageManifest)
	if err != nil {
		return err
	}

	// Download package file.
	err = r.downloadFile(remote.fileURL, packagesRoot, packageFile)
	if err != nil {
		return err
	}

	return nil
}

// getRemotePackageInfoInGithub checks that the given package is available in the remote
// repository. In order to confirm the package really exists, both manifest
// and the actual package content must exist in remote repository.
func (r *Repo) getRemotePackageInfoInGithub(name string) (*RemotePackageDownloadInfo, error) {
	// Get file listing for the remote repository.
	releases, err := r.queryReleases()
	if err != nil {
		return nil, err
	}

	// Walk release by release until you find one that has both manifest and file asset
	for _, release := range releases {
		info := RemotePackageDownloadInfo{}
		for _, asset := range release.Assets {
			if asset.Name == (name + ".yaml") {
				info.manifestURL = asset.DownloadUrl
			}
			if asset.Name == (name + ".mpm") || asset.Name == (name + ".mpm.x86_64") {
				info.fileURL = asset.DownloadUrl
			}

			// Both must be found for package to exist in remote repository.
			if info.manifestURL != "" && info.fileURL != "" {
				return &info, nil
			}
		}
	}
	return nil, nil
}

func (r *Repo) githubPackageInfoRemote(packageName string) *core.Package {
	packageDownloadInfo, err := r.getRemotePackageInfoInGithub(packageName)
	if err != nil {
		return nil
	}

	return remotePackageInfo(packageDownloadInfo.manifestURL)
}

func (r *Repo) githubMakeReleaseApiCall(suffix string) ([]byte, error) {
	var	netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	fullUrl := r.GithubURL + OsvReleasesSuffix + suffix
	resp, err := netClient.Get(fullUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("The request %s returned non-200 [%d] response: %s.",
			fullUrl, resp.StatusCode, string(body))
	}
	return body, nil
}
