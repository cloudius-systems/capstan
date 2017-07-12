/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package testing

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//
// Common File Content Templates
//

const PackageYamlText string = `
name: package-name
title: PackageTitle
author: package-author
`

const DefaultText string = `
Some text.
`

// PrepareFiles realizes map[filepath]content into given directory.
// E.g. directory = /tmp/sample, files = {"/file01.txt" => "Foo Bar"} will result in
//
// /tmp/sample/
//        |- file01.txt
//
// where file01.txt will contain text content of "Foo Bar".
func PrepareFiles(directory string, files map[string]string) error {
	for path, content := range files {
		path = strings.TrimPrefix(path, "/")

		// Create directory structure.
		if err := os.MkdirAll(filepath.Join(directory, filepath.Dir(path)), 0700); err != nil {
			return err
		}
		// Create file with content.
		if err := ioutil.WriteFile(filepath.Join(directory, path), []byte(content), 0700); err != nil {
			return err
		}
	}

	return nil
}
