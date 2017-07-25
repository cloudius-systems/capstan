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
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

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

// DirEquals checker checks that given directory contains exactly given files
// with given content.
//
// For example:
//
//     c.Assert("/tmp/mydir", DirEquals, map[string]string{"/file01.txt": "Exact content"})
//
var DirEquals Checker = &dirEqualsChecker{
	&CheckerInfo{Name: "DirEquals", Params: []string{"obtained", "expected"}},
}

type dirEqualsChecker struct {
	*CheckerInfo
}

func (checker *dirEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	defer func() {
		if v := recover(); v != nil {
			result = false
			error = fmt.Sprint(v)
		}
	}()

	dirPath, ok := params[0].(string)
	if !ok {
		return false, "Obtained value must be a path to directory"
	}

	// Open and loop directory.
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return false, fmt.Sprintf("Obtained value must be a path to directory: %s", err.Error())
	}
	obtained := map[string]string{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(dirPath, f.Name()))
		if err != nil {
			return false, fmt.Sprintf("failed to read file '%s' from directory: %s", f.Name(), err.Error())
		}
		obtained[f.Name()] = string(data)
	}

	isOk := reflect.DeepEqual(obtained, params[1])

	// When match is false, we show user the content, not the filepath.
	if !isOk {
		params[0] = obtained
	}

	return isOk, ""
}

// The MatchesMultiline checker verifies that the string provided as the obtained
// value (or the string resulting from obtained.String()) matches the
// regular expression provided and is matched against multiline string.
//
// For example:
//
//     c.Assert(v, Matches, "perm.*denied")
//
var MatchesMultiline Checker = &matchesMultilineChecker{
	&CheckerInfo{Name: "Matches", Params: []string{"value", "regex"}},
}

type matchesMultilineChecker struct {
	*CheckerInfo
}

func (checker *matchesMultilineChecker) Check(params []interface{}, names []string) (result bool, error string) {
	return matchesMultiline(params[0], params[1])
}

func matchesMultiline(value, regex interface{}) (result bool, error string) {
	reStr, ok := regex.(string)
	if !ok {
		return false, "Regex must be a string"
	}
	valueStr, valueIsStr := value.(string)
	if !valueIsStr {
		if valueWithStr, valueHasStr := value.(fmt.Stringer); valueHasStr {
			valueStr, valueIsStr = valueWithStr.String(), true
		}
	}
	if valueIsStr {
		matches, err := regexp.MatchString(reStr, valueStr)
		if err != nil {
			return false, "Can't compile regex: " + err.Error()
		}
		return matches, ""
	}
	return false, "Obtained value is not a string and has no .String()"
}

// The BootCmdEquals checker verifies that the bootCmd string (possibly with --env prefixes)
// matches the expected boot command and environment variables. Three arguments are required
// to the right of the checker:
// - bootCmd string that contains only the bootcmd without --env prefixes
// - env map[string]string that defines expected environment variables
// - soft bool that switches between '=' (when false) and '?=' (when true) operator
//
// For example:
//
//     c.Assert("--env=A=1 /node server.js", BootCmdEquals, "/node server.js", map[string]string{"A": "1"}, true)
//
var BootCmdEquals Checker = &bootCmdEqualsChecker{
	&CheckerInfo{Name: "BootCmdEquals", Params: []string{"obtained", "bootcmd", "env"}},
}

type bootCmdEqualsChecker struct {
	*CheckerInfo
}

func (checker *bootCmdEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	obtained, ok := params[0].(string)
	if !ok {
		return false, "Obtained value must be a string."
	}
	bootCmd, ok := params[1].(string)
	if !ok {
		return false, "First expected value must be a string."
	}
	env, ok := params[2].([]string)
	if !ok {
		return false, "Second expected value must be a list of expected --env prefixes."
	}

	if err := CheckBootCmd(obtained, bootCmd, env); err == nil {
		return true, ""
	} else {
		return false, err.Error()
	}
}

func CheckBootCmd(obtained, bootCmd string, env []string) error {
	for _, val := range env {
		envString := fmt.Sprintf("%s ", val)
		if strings.Contains(obtained, envString) {
			obtained = strings.Replace(obtained, envString, "", 1)
		} else {
			return fmt.Errorf("missing '%s'", envString)
		}
	}

	if obtained != bootCmd {
		return fmt.Errorf("bootcmd '%s' does not equal '%s'", obtained, bootCmd)
	}

	return nil
}
