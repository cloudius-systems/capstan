/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Capstanignore interface {
	LoadFile(path string) error
	AddPattern(pattern string) error
	PrintPatterns()
	IsIgnored(path string) bool
}

var CAPSTANIGNORE_ALWAYS []string = []string{"/meta/*", "/mpm-pkg/*", "/.git/*"}

// CapstanignoreInit creates a new capstanignore struct that is
// used when deciding whether a file should be included in unikernel
// or not. You can provide `path` to the .capstanignore file to load
// it or pass empty string "" if you have none. Note that once having
// capstanignore struct you can load as many files as you want (using
// .LoadFile function) or manually add as many patterns as you like
// (using .AddPattern function).
func CapstanignoreInit(path string) Capstanignore {
	c := capstanignore{}

	// Load capstanignore file if path was given.
	if path != "" {
		if err := c.LoadFile(path); err != nil {
			fmt.Println("WARN: failed to load .capstanignore file:", err)
		}
	}

	// Always ignore some common paths.
	for _, pattern := range CAPSTANIGNORE_ALWAYS {
		c.AddPattern(pattern)
	}
	return &c
}

type capstanignore struct {
	ignored  []string         // list of all ignored patterns
	ignoredC []*regexp.Regexp // list of compiled patterns
}

// LoadFile attempts to parse .capstanignore file on given path.
// If success, it remembers all patterns and closes file.
func (c *capstanignore) LoadFile(path string) error {
	if file, err := os.Open(path); err == nil {
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.Trim(scanner.Text(), " ")
			if len(line) == 0 || strings.HasPrefix(line, "#") {
				continue
			}
			if errPattern := c.AddPattern(line); errPattern != nil {
				return errPattern
			}
		}
	} else {
		return err
	}
	return nil
}

// AddPattern adds a pattern to be ignored.
func (c *capstanignore) AddPattern(pattern string) error {
	safePattern := transformCapstanignoreToRegex(pattern)
	if compiled, err := regexp.Compile(safePattern); err == nil {
		c.ignored = append(c.ignored, pattern)
		c.ignoredC = append(c.ignoredC, compiled)
	} else {
		return err
	}

	// also ignore folder when all its content is ignored
	for strings.HasSuffix(pattern, "/*") {
		pattern = strings.TrimSuffix(pattern, "/*")
		safePattern = transformCapstanignoreToRegex(pattern)
		compiled := regexp.MustCompile(safePattern)
		c.ignoredC = append(c.ignoredC, compiled)
	}

	return nil
}

func (c *capstanignore) IsIgnored(path string) bool {
	for _, pattern := range c.ignoredC {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

func (c *capstanignore) PrintPatterns() {
	for _, pattern := range c.ignored {
		fmt.Println(pattern)
	}
}

// transformCapstanignoreToRegex transforms capstanignore synstax to regex systax.
func transformCapstanignoreToRegex(pattern string) string {
	// preprocess
	pattern = strings.Replace(pattern, "/**/", "{two-stars}", -1)
	if strings.HasSuffix(pattern, "/*") {
		pattern = strings.TrimSuffix(pattern, "/*")
		pattern = pattern + "{all-beneath}"
	}

	// Implicit ^ at the beginning
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	// Star * means only one folder level
	pattern = strings.Replace(pattern, "*", "[^/]*", -1)
	// Dot . means actual dot
	pattern = strings.Replace(pattern, ".", "\\.", -1)
	// Two stars between two slashes /**/ mean all folder levels
	pattern = strings.Replace(pattern, "{two-stars}", ".*", -1)
	// /* at the end means also all subfolders
	pattern = strings.Replace(pattern, "{all-beneath}", "/.*", 1)
	// Implicit $ at the end
	if !strings.HasSuffix(pattern, "$") {
		pattern = pattern + "$"
	}

	return pattern
}
