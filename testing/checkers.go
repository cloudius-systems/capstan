/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package testing

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"

	. "gopkg.in/check.v1"
)

// TarGzEquals checker checks that given tar.gz archive contains exactly given files
// with given content.
//
// For example:
//
//     c.Assert("/tmp/archive.tar.gz", TarGzEquals, map[string]string{"/file01.txt": "Exact content"})
//
var TarGzEquals Checker = &tarGzEqualsChecker{
	&CheckerInfo{Name: "TarGzEquals", Params: []string{"obtained", "expected"}},
}

type tarGzEqualsChecker struct {
	*CheckerInfo
}

func (checker *tarGzEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	defer func() {
		if v := recover(); v != nil {
			result = false
			error = fmt.Sprint(v)
		}
	}()

	tarGzPath, ok := params[0].(string)
	if !ok {
		return false, "Obtained value must be a path to tar.gz file"
	}

	// Open and read tar.gz file.
	f, err := os.Open(tarGzPath)
	if err != nil {
		return false, fmt.Sprintf("Obtained value must be a path to tar.gz file: %s", err.Error())
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return false, fmt.Sprintf("failed to decompress tar.gz: %s", err.Error())
	}
	tarReader := tar.NewReader(gzReader)

	obtained := map[string]string{}
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				// Have we reached till the end of the tar?
				break
			}
			return false, fmt.Sprintf("failed to parse tar: %s", err.Error())
		}

		// This checker only focuses on files, for the sake of simplicity.
		if header.FileInfo().IsDir() {
			continue
		}

		data, err := ioutil.ReadAll(tarReader)
		if err != nil {
			return false, fmt.Sprintf("failed to read file '%s' from tar: %s", header.Name, err.Error())
		}

		obtained[header.Name] = string(data)
	}

	isOk := reflect.DeepEqual(obtained, params[1])

	// When match is false, we show user the content, not the filepath.
	if !isOk {
		params[0] = obtained
	}

	return isOk, ""
}
