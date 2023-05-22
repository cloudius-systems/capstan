/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"encoding/xml"
	"fmt"
	"github.com/cloudius-systems/capstan/core"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultRepositoryUrl = "https://mikelangelo-capstan.s3.amazonaws.com/"
)

type Contents struct {
	Key          string
	LastModified string
	Size         int
	StorageClass string
}

type Query struct {
	ContentsList []Contents `xml:"Contents"`
}

func queryRemote(repo_url string) (*Query, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := netClient.Get(repo_url)
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
			repo_url, resp.StatusCode, string(body))
	}
	var q Query
	xml.Unmarshal(body, &q)
	return &q, nil
}

func ListImagesRemote(repo_url string, search string) error {
	q, err := queryRemote(repo_url)
	if err != nil {
		return err
	}
	fmt.Println(FileInfoHeader())
	for _, content := range q.ContentsList {
		if strings.HasSuffix(content.Key, "index.yaml") {
			if img := RemoteFileInfo(repo_url, content.Key); img != nil && strings.Contains(img.Name, search) {
				fmt.Println(img.String())
			}
		}
	}
	return nil
}

func (r *Repo) DownloadImage(hypervisor string, path string) error {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return fmt.Errorf("%s: wrong name format", path)
	}
	err := os.MkdirAll(filepath.Join(r.RepoPath(), path), os.ModePerm)
	if err != nil {
		return err
	}
	fileName := fmt.Sprintf("%s/index.yaml", path)
	err = r.downloadFile(r.URL+fileName, r.RepoPath(), fileName)
	if err != nil {
		return err
	}
	fileName = fmt.Sprintf("%s/%s.%s.gz", path, parts[1], hypervisor)
	return r.downloadFile(r.URL+fileName, r.RepoPath(), fileName)
}

func IsRemoteImage(repo_url, name string) (bool, error) {
	q, err := queryRemote(repo_url)
	if err != nil {
		return false, err
	}
	for _, content := range q.ContentsList {
		if strings.HasPrefix(content.Key, name+"/") {
			return true, nil
		}
	}
	return false, nil
}

func (r *Repo) downloadLoaderImageFromS3(hypervisor string) (string, error) {
	imageName := LoaderImageName
	err := r.DownloadImage(hypervisor, imageName)

	if err == nil {
		fmt.Printf("Downloaded loader image (%s) from S3.\n", imageName)
		return imageName, nil
	} else {
		return imageName, err
	}
}

// DownloadPackage downloads a package from the S3 repository into local.
func (r *Repo) downloadPackageFromS3(packageName string) error {
	remote, err := r.getRemotePackageInfoInS3(packageName)
	if err != nil {
		return err
	}
	// If the package is not found on a remote repository, inform the user.
	if remote == nil {
		return fmt.Errorf("package %s is not available in the given repository (%s)", packageName, r.URL)
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
	err = r.downloadFile(r.URL+"packages/"+packageManifest, packagesRoot, packageManifest)
	if err != nil {
		return err
	}

	// Download package file.
	err = r.downloadFile(r.URL+"packages/"+packageFile, packagesRoot, packageFile)
	if err != nil {
		return err
	}

	return nil
}

// IsRemotePackage checks that the given package is available in the remote
// repository. In order to confirm the package really exists, both manifest
// and the actual package content must exist in remote repository.
func (r *Repo) getRemotePackageInfoInS3(name string) (*RemotePackageDownloadInfo, error) {
	// Get file listing for the remote repository.
	q, err := queryRemote(r.URL)
	if err != nil {
		return nil, err
	}

	info := RemotePackageDownloadInfo{}

	for _, content := range q.ContentsList {
		if strings.HasPrefix(content.Key, "packages/") {
			// Check whether the current file is either package manifest or content file.
			if strings.HasSuffix(content.Key, name+".yaml") {
				info.manifestURL = r.URL + content.Key
			} else if strings.HasSuffix(content.Key, name+".mpm") {
				info.fileURL = r.URL + content.Key
			}

			// Both must be found for package to exist in remote repository.
			if info.manifestURL != "" && info.fileURL != "" {
				return &info, nil
			}
		}
	}

	return nil, nil
}

func (r *Repo) s3ListPackagesRemote(search string) error {
	q, err := queryRemote(r.URL)
	if err != nil {
		return err
	}
	fmt.Println(FileInfoHeader())
	for _, content := range q.ContentsList {
		if strings.HasPrefix(content.Key, "packages/") && strings.HasSuffix(content.Key, ".yaml") {
			if pkg := remotePackageInfo(r.URL + content.Key); pkg != nil && strings.Contains(pkg.Name, search) {
				fmt.Println(pkg.String())
			}
		}
	}
	return nil
}

func (r *Repo) s3PackageInfoRemote(packageName string) *core.Package {
	return remotePackageInfo(r.URL + fmt.Sprintf("packages/%s.yaml", packageName))
}
