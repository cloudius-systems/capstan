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

const bucket_url = "http://osv.capstan.s3.amazonaws.com/"

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
	return fmt.Sprintf("%-30s %-50s %-25s %-15s", "Name", "Description", "Version", "Created")
}

func (f *FileInfo) String() string {
	return fmt.Sprintf("%-30s %-50s %-25s %-15s", f.Namespace+"/"+f.Name, f.Description, f.Version, f.Created)
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

func RemoteFileInfo(path string) *FileInfo {
	resp, err := http.Get(bucket_url + path)
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

func QueryRemote() *Query {
	resp, err := http.Get(bucket_url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var q Query
	xml.Unmarshal(body, &q)
	return &q
}

func ListImagesRemote(search string) {
	fmt.Println(FileInfoHeader())
	q := QueryRemote()
	for _, content := range q.ContentsList {
		if strings.HasSuffix(content.Key, "index.yaml") {
			if img := RemoteFileInfo(content.Key); img != nil && strings.Contains(img.Name, search) {
				fmt.Println(img.String())
			}
		}
	}
}

func (r *Repo) DownloadFile(name string) error {
	compressed := strings.HasSuffix(name, ".gz")
	output, err := os.Create(filepath.Join(r.Path, strings.TrimSuffix(name, ".gz")))
	if err != nil {
		return err
	}
	defer output.Close()
	fmt.Printf("Downloading %s...\n", name)
	tr := &http.Transport{
		DisableCompression: true,
                Proxy: http.ProxyFromEnvironment,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(bucket_url + name)
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

func (r *Repo) DownloadImage(hypervisor string, path string) error {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return fmt.Errorf("%s: wrong name format", path)
	}
	err := os.MkdirAll(filepath.Join(r.Path, path), os.ModePerm)
	if err != nil {
		return err
	}
	err = r.DownloadFile(fmt.Sprintf("%s/index.yaml", path))
	if err != nil {
		return err
	}
	return r.DownloadFile(fmt.Sprintf("%s/%s.%s.gz", path, parts[1], hypervisor))
}

func IsRemoteImage(name string) bool {
	q := QueryRemote()
	for _, content := range q.ContentsList {
		if strings.HasPrefix(content.Key, name+"/") {
			return true
		}
	}
	return false
}
