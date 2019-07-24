package util

import (
	"compress/gzip"
	"fmt"
	"github.com/cheggaaa/pb"
	"github.com/cloudius-systems/capstan/core"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileInfo struct {
	Namespace   string
	Name        string
	Description string
	Version     string
	Created     core.YamlTime `yaml:"created"`
	Platform    string
}

type FilesInfo struct {
	images []FileInfo
}

type RemotePackageDownloadInfo struct {
	manifestURL string
	fileURL     string
}

func FileInfoHeader() string {
	res := fmt.Sprintf("%-50s %-50s %-15s %-20s %-15s", "Name", "Description", "Version", "Created", "Platform")
	return strings.TrimSpace(res)
}

func (f *FileInfo) String() string {
	// Trim "/" prefix if there is one (happens when namespace is empty)
	name := strings.TrimLeft(f.Namespace+"/"+f.Name, "/")
	platform := f.Platform
	if platform == "" {
		platform = "N/A"
	}
	res := fmt.Sprintf("%-50s %-50s %-15s %-20s %-15s", name, f.Description, f.Version, f.Created, platform)
	return strings.TrimSpace(res)
}

func ParseIndexYaml(path, ns, name string) (*FileInfo, error) {
	data, err := ioutil.ReadFile(filepath.Join(path, ns, name, "index.yaml"))
	if err != nil {
		return nil, err
	}
	f := FileInfo{}
	err = yaml.Unmarshal(data, &f)
	if err != nil {
		return nil, err
	}
	f.Namespace = ns
	f.Name = name
	return &f, nil
}

func RemoteFileInfo(repo_url string, path string) *FileInfo {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := netClient.Get(repo_url + path)
	if err != nil {
		return nil
	}

	parts := strings.Split(path, "/")
	defer resp.Body.Close()
	f := FileInfo{}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	if resp.StatusCode != 200 {
		fmt.Printf("The request %s returned non-200 [%d] response: %s.",
			repo_url+path, resp.StatusCode, string(data))
		return nil
	}
	err = yaml.Unmarshal(data, &f)
	if err != nil {
		return nil
	}
	if err != nil {
		return nil
	}
	f.Namespace = parts[0]
	f.Name = parts[1]
	return &f
}

// remotePackageInfo downloads the given manifest files and tries to parse it.
// core.Package struct is returned if it succeeds, otherwise nil.
func remotePackageInfo(package_url string) *core.Package {
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := netClient.Get(package_url)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Printf("The request %s returned non-200 [%d] response: %s.",
			package_url, resp.StatusCode, string(data))
		return nil
	}

	var pkg core.Package

	if err := pkg.Parse(data); err != nil {
		return nil
	}

	return &pkg
}

func NeedsUpdate(localPkg, remotePkg *core.Package, compareCreated bool) (bool, error) {
	// Compare Version attribute.
	localVersion, err := VersionStringToInt(localPkg.Version)
	if err != nil {
		return true, err
	}
	remoteVersion, err := VersionStringToInt(remotePkg.Version)
	if err != nil {
		return true, err
	}
	needsUpdate := localVersion < remoteVersion
	if needsUpdate || !compareCreated {
		return needsUpdate, nil
	}

	// Compare Created attribute.
	createdLocal := localPkg.Created.GetTime()
	createdRemote := remotePkg.Created.GetTime()
	if createdLocal == nil || createdRemote == nil {
		return true, nil
	}
	return createdLocal.Before(*createdRemote), nil
}

func (r *Repo) downloadFile(fileURL string, destPath string, name string) error {
	compressed := strings.HasSuffix(fileURL, ".gz")
	output, err := os.Create(filepath.Join(destPath, strings.TrimSuffix(name, ".gz")))
	if err != nil {
		return err
	}
	defer output.Close()
	fmt.Printf("Downloading %s... from %s\n", name, fileURL)
	tr := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bar := pb.New64(resp.ContentLength).SetUnits(pb.U_BYTES)
	bar.Start()
	proxyReader := bar.NewProxyReader(resp.Body)
	var reader io.Reader = proxyReader
	if compressed {
		gzipReader, err := gzip.NewReader(proxyReader)
		if err != nil {
			return err
		}
		reader = gzipReader
	}
	_, err = io.Copy(output, reader)
	bar.Finish()
	if err != nil {
		return err
	}
	return nil
}
