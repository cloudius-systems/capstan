/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudius-systems/capstan/hypervisor"
	"github.com/cloudius-systems/capstan/hypervisor/qemu"
)

// Volume is an extended version of hypervisor.Volume with volume creation data.
type Volume struct {
	hypervisor.Volume
	SizeMB int64
	Name   string
}

const (
	VOLUMES_DIR           string = "volumes"
	VOLUME_FORMAT_DEFAULT string = "raw"
)

// CreateVolume creates volume with given specifications.
func CreateVolume(packagePath string, volume Volume) error {
	if _, err := os.Stat(filepath.Join(packagePath, "meta", "package.yaml")); os.IsNotExist(err) {
		return fmt.Errorf("Must be in package root directory")
	}
	volumesDir := filepath.Join(packagePath, VOLUMES_DIR)
	if _, err := os.Stat(volumesDir); os.IsNotExist(err) {
		if err := os.Mkdir(volumesDir, 0744); err != nil {
			return fmt.Errorf("Could not create volumes dir '%s': %s", volumesDir, err)
		}
	}

	// Set defaults
	if volume.Format == "" {
		volume.Format = VOLUME_FORMAT_DEFAULT
	}

	// Create actual volume.
	volume.Path = filepath.Join(volumesDir, fmt.Sprintf(volume.Name))
	if err := qemu.CreateVolume(volume.Path, volume.Format, volume.SizeMB); err != nil {
		return fmt.Errorf("Could not create volume: %s", err)
	}

	// Write metadata.
	if err := volume.PersistMetadata(); err != nil {
		return fmt.Errorf("Could not persist volume metadata: %s", err)
	}

	return nil
}

// DeleteVolume deletes volume and its metadata with given name.
func DeleteVolume(packagePath, name string, verbose bool) error {
	path := filepath.Join(packagePath, VOLUMES_DIR, name)
	meta := fmt.Sprintf("%s.yaml", path)

	// Remove the volume itself.
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err == nil {
			if verbose {
				fmt.Println("Removed volume", path)
			}
		} else {
			return err
		}
	} else {
		return fmt.Errorf("Could not find volume with name '%s'", name)
	}

	// Remove the metadata if it exists.
	if _, err := os.Stat(meta); err == nil {
		if err := os.Remove(meta); err == nil {
			if verbose {
				fmt.Println("Removed volume metadata", meta)
			}
		} else {
			return err
		}
	}

	return nil
}
