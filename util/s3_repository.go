/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"github.com/cheggaaa/pb"
	"gopkg.in/yaml.v1"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Namespace   string
	Name        string
	Description string
	Version     string
	Created     string
}

type Contents struct {
	Key          string
	LastModified string
	Size         int
	StorageClass string
}

type Query struct {
	ContentsList []Contents `xml:"Contents"`
}

type FilesInfo struct {
	images []FileInfo
}

func FileInfoHeader() string {
	return fmt.Sprintf("%-50s %-50s %-25s %-15s", "Name", "Description", "Version", "Created")
}

func (f *FileInfo) String() string {
	return fmt.Sprintf("%-50s %-50s %-25s %-15s", f.Namespace+"/"+f.Name, f.Description, f.Version, f.Created)
}

func MakeFileInfo(path, ns, name string) *FileInfo {
	data, err := ioutil.ReadFile(filepath.Join(path, ns, name, "index.yaml"))
	if err != nil {
		return nil
	}
	f := FileInfo{}
	err = yaml.Unmarshal(data, &f)
	if err != nil {
		return nil
	}
	f.Namespace = ns
	f.Name = name
	return &f
}

func RemoteFileInfo(repo_url string, path string) *FileInfo {
	resp, err := http.Get(repo_url + path)
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

func QueryRemote(repo_url string) (*Query, error) {
	resp, err := http.Get(repo_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var q Query
	xml.Unmarshal(body, &q)
	return &q, nil
}

func ListImagesRemote(repo_url string, search string) error {
	q, err := QueryRemote(repo_url)
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

func (r *Repo) DownloadFile(repo_url string, name string) error {
	compressed := strings.HasSuffix(name, ".gz")
	output, err := os.Create(filepath.Join(r.RepoPath(), strings.TrimSuffix(name, ".gz")))
	if err != nil {
		return err
	}
	defer output.Close()
	fmt.Printf("Downloading %s...\n", name)
	tr := &http.Transport{
		DisableCompression: true,
		Proxy:              http.ProxyFromEnvironment,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(repo_url + name)
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

func (r *Repo) DownloadImage(repo_url, hypervisor string, path string) error {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return fmt.Errorf("%s: wrong name format", path)
	}
	err := os.MkdirAll(filepath.Join(r.RepoPath(), path), os.ModePerm)
	if err != nil {
		return err
	}
	err = r.DownloadFile(repo_url, fmt.Sprintf("%s/index.yaml", path))
	if err != nil {
		return err
	}
	return r.DownloadFile(repo_url, fmt.Sprintf("%s/%s.%s.gz", path, parts[1], hypervisor))
}

func IsRemoteImage(repo_url, name string) (bool, error) {
	q, err := QueryRemote(repo_url)
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
