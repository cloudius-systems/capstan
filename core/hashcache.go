/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package core

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type HashCache map[string]string

func NewHashCache() HashCache {
	return make(map[string]string)
}

// ParseHashCache looks for a file at given location and tries to
// parse the HashCache config. In case the file does not exist
// or is not a valid HashCache file, it fails with an error.
func ParseHashCache(cachePath string) (HashCache, error) {
	var hc HashCache

	// Make sure the cache file exists.
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return hc, err
	}

	// Read the cache file.
	d, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return hc, err
	}

	// And parse it.
	if err := hc.parse(d); err != nil {
		return hc, err
	}

	return hc, nil
}

func (h *HashCache) parse(data []byte) error {
	if err := yaml.Unmarshal(data, h); err != nil {
		return err
	}

	return nil
}

func (h *HashCache) WriteToFile(path string) error {
	data, err := yaml.Marshal(h)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
