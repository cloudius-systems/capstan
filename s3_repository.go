/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package capstan

import (
	"encoding/xml"
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const time_layout = "2006-01-02T15:04:05"
const bucket_url = "http://osv.capstan.s3.amazonaws.com/"

type FileInfo struct {
	namespace   string
	name        string
	description string
	version     string
	created     time.Time
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
	return fmt.Sprintf("%-15s %-30s %-15s %-15s", "Name", "Description", "Version", "Created")
}

func (f *FileInfo) String() string {
	return fmt.Sprintf("%-15s %-30s %-15s %-15s", f.namespace+"/"+f.name, f.description, f.version, f.created.Format(time_layout))
}

func yamlToInfo(ns, name string, mp yaml.Node) *FileInfo {
	if mp == nil {
		return nil
	}
	m := mp.(yaml.Map)
	y2s := func(key string) string {
		return m[key].(yaml.Scalar).String()
	}

	tm, _ := time.Parse(time_layout, strings.TrimSpace(y2s("created")))
	return &FileInfo{namespace: ns, name: name, version: y2s("version"), created: tm, description: y2s("description")}
}

func MakeFileInfo(path, ns, name string) *FileInfo {
	file, err := yaml.ReadFile(filepath.Join(path, ns, name, "index.yaml"))
	if err != nil {
		return nil
	}
	return yamlToInfo(ns, name, file.Root)
}

func RemoteFileInfo(path string) *FileInfo {
	resp, err := http.Get(bucket_url + path)
	if err != nil {
		return nil
	}

	parts := strings.Split(path, "/")
	defer resp.Body.Close()
	result, err := yaml.Parse(resp.Body)
	if err != nil {
		return nil
	}
	return yamlToInfo(parts[0], parts[1], result)
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
			if img := RemoteFileInfo(content.Key); img != nil && strings.Contains(img.name, search) {
				fmt.Println(img.String())
			}
		}
	}
}

func (r *Repo) DownloadFile(name string) error {
	output, err := os.Create(filepath.Join(r.Path, strings.TrimSuffix(name,".gz")))
	if err != nil {
		return err
	}
	defer output.Close()
	fmt.Printf("Fetching %s...", name)
	resp, err := http.Get(bucket_url + name)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	n, err := io.Copy(output, resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(n, "bytes downloaded.")
	return nil
}

func (r *Repo) DownloadImage(path string) error {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {		
		return fmt.Errorf("%s: wrong name format", path)
	}
	q := QueryRemote()
	for _, content := range q.ContentsList {
		if strings.HasPrefix(content.Key, path+"/") && content.Size > 0 {
			os.MkdirAll(filepath.Join(r.Path,path), os.ModePerm)
			err := r.DownloadFile(content.Key)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
