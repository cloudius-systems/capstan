/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package capstan

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const time_layout = "2006-01-02T15:04:05"
const bucket_url = "http://cpastan01.amnon.osv.s3.amazonaws.com/"

var headerWidth = []int{15, 30, 15, 15, 0, 0}

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

func strWidth(str string, width int) string {
	return fmt.Sprintf("%-"+strconv.Itoa(width)+"s", str)
}

func FileInfoFmt(vals []string) string {
	var buffer bytes.Buffer
	for i := 0; i < len(vals); i++ {
		if headerWidth[i] > 0 {
			buffer.WriteString(strWidth(vals[i], headerWidth[i]))
		}
	}
	return buffer.String()
}

func FileInfoHeader() string {
	vals := []string{"Name", "Description", "Version", "Created", "Hypervisor", "Extension"}
	return FileInfoFmt(vals)
}

func (f *FileInfo) String() string {
	vals := []string{f.namespace + "/" + f.name, f.description, f.version, f.created.Format(time_layout)}
	return FileInfoFmt(vals)
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
	output, err := os.Create(filepath.Join(r.Path, name))
	if err != nil {
		return err
	}
	defer output.Close()
	resp, err := http.Get(bucket_url + name)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	n, err := io.Copy(output, resp.Body)
	if err != nil {
		fmt.Println("Error while downloading", bucket_url+name, "-", err)
		return err
	}
	fmt.Println(n, "bytes downloaded.")
	return nil
}

func (r *Repo) DownloadImage(path string) error {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		errors.New(fmt.Sprintf("%s: wrong name format", path))
	}
	q := QueryRemote()
	for _, content := range q.ContentsList {
		if strings.HasPrefix(content.Key+"/", path) && content.Size > 0 {
			r.DownloadFile(content.Key)
		}
	}
	return nil
}

func IsRemoteImage(name string) bool {
	q := QueryRemote()
	for _, content := range q.ContentsList {
		if content.Key == name+"/" {
			return true
		}
	}
	return false
}
