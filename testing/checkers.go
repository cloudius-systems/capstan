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
	"regexp"
	"strings"

	. "gopkg.in/check.v1"
)

// TarGzEquals checker checks that given tar.gz archive contains exactly given files
// with given content. Argument 'expected' must be a map[string]interface{} where
// key is path of the file and value is one of type:
// * string           - file content must match exactly for checker to succed
// * func(string)error- file content is passed to this function and it must return nil
//                      for checker to succeed
//
// For example:
//
//     c.Assert("/tmp/archive.tar.gz", TarGzEquals, map[string]string{"/file01.txt": "Exact content"})
//     c.Assert("/tmp/archive.tar.gz", TarGzEquals, map[string]string{"/file01.txt": func(val string)error{return nil}})
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

	expected, ok := params[1].(map[string]interface{})
	if !ok {
		return false, "Expected value must be map[string]interface{}"
	}

	obtained, err := loadTarGz(tarGzPath)
	if err != nil {
		return false, err.Error()
	}

	// Compare.
	if err := compareMaps(obtained, expected); err != nil {
		// When match is false, we show user the content, not the filepath.
		params[0] = obtained

		return false, err.Error()
	}

	return true, ""
}

func loadTarGz(tarGzPath string) (map[string]string, error) {
	// Open and read tar.gz file.
	f, err := os.Open(tarGzPath)
	if err != nil {
		return nil, fmt.Errorf("Obtained value must be a path to tar.gz file: %s", err.Error())
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress tar.gz: %s", err.Error())
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
			return nil, fmt.Errorf("failed to parse tar: %s", err.Error())
		}

		// This checker only focuses on files, for the sake of simplicity.
		if header.FileInfo().IsDir() {
			continue
		}

		data, err := ioutil.ReadAll(tarReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s' from tar: %s", header.Name, err.Error())
		}

		obtained[header.Name] = string(data)
	}

	return obtained, nil
}

// DirEquals checker checks that given directory contains exactly given files
// with given content. Argument 'expected' must be a map[string]interface{} where
// key is path of the file and value is one of type:
// * string           - file content must match exactly for checker to succed
// * func(string)error- file content is passed to this function and it must return nil
//                      for checker to succeed
//
// For example:
//
//     c.Assert("/tmp/mydir", DirEquals, map[string]string{"/file01.txt": "Exact content"})
//     c.Assert("/tmp/mydir", DirEquals, map[string]string{"/file01.txt": func(val string)error{return nil}})
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

	expected, ok := params[1].(map[string]interface{})
	if !ok {
		return false, "Expected value must be map[string]interface{}"
	}

	obtained, err := loadDir(dirPath)
	if err != nil {
		return false, err.Error()
	}

	// Compare.
	if err := compareMaps(obtained, expected); err != nil {
		// When match is false, we show user the content, not the filepath.
		params[0] = obtained

		return false, err.Error()
	}

	return true, ""
}

func loadDir(dirPath string) (map[string]string, error) {
	// Open and loop directory.
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Obtained value must be a path to directory: %s", err.Error())
	}
	obtained := map[string]string{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(dirPath, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s' from directory: %s", f.Name(), err.Error())
		}
		obtained[f.Name()] = string(data)
	}
	return obtained, nil
}

func compareMaps(obtained map[string]string, expected map[string]interface{}) error {
	if len(obtained) != len(expected) {
		return fmt.Errorf("obtained map key is not as expected")
	}

	for key, val := range expected {
		_, exists := obtained[key]
		if !exists {
			return fmt.Errorf("expected key '%s' not found in obtained map", key)
		}

		if expectedString, ok := val.(string); ok {
			if expectedString != obtained[key] {
				return fmt.Errorf("mismatch for key '%s': '%s' != '%s'", key, val, obtained[key])
			}
		} else if expectedFn, ok := val.(func(string) error); ok {
			if err := expectedFn(obtained[key]); err != nil {
				return fmt.Errorf("mismatch for key '%s': %s", key, err.Error())
			}
		} else {
			return fmt.Errorf("Invalid expectation for key '%s'", key)
		}
	}
	return nil
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
