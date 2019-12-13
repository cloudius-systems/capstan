/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/cloudius-systems/capstan/core"
	"github.com/cloudius-systems/capstan/cpio"
	"github.com/cloudius-systems/capstan/hypervisor/qemu"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/util"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func Compose(r *util.Repo, loaderImage string, imageSize int64, uploadPath string, appName string, commandLine string, verbose bool) error {
	// Initialize an empty image based on the provided loader image. imageSize is used to
	// determine the size of the user partition.
	err := r.InitializeZfsImage(loaderImage, appName, imageSize)
	if err != nil {
		return err
	}

	// Get the path of imported image.
	imagePath := r.ImagePath("qemu", appName)

	paths, err := CollectPathContents(uploadPath)
	if err != nil {
		return err
	}

	// Upload the specified path onto virtual image.
	if _, err = UploadPackageContents(r, imagePath, paths, nil, verbose); err != nil {
		return err
	}

	if commandLine != "" {
		// Set the command line.
		if err = util.SetCmdLine(imagePath, commandLine); err != nil {
			return err
		}
		fmt.Printf("Command line set to: '%s'\n", commandLine)
	}

	return nil
}

func UploadPackageContents(r *util.Repo, appImage string, uploadPaths map[string]string, imageCache core.HashCache, verbose bool) (core.HashCache, error) {

	var osvCmdline string

	if len(imageCache) == 0 {
		fmt.Printf("Uploading files to %s...\n", appImage)
		// It is asumed that the UploadPath is the first command executed by
		// this virtual image.  Thus we also create the filesystem and start
		// the 'cpiod' daemon responsible for copying files to target VM.
		osvCmdline = "--norandom --nomount --noinit /tools/mkfs.so; /tools/cpiod.so --prefix /zfs/zfs; /zfs.so set compression=off osv"
	} else {
		fmt.Printf("Updating image %s...\n", appImage)
		// If we are updating an existing image, we should only start cpiod
		// allowing us to upload modified files. Files are always uploaded onto
		// root
		osvCmdline = "/tools/cpiod.so --prefix /"
	}

	// Specify the VM properties. Use the app image as the source to start.
	vmconfig := &qemu.VMConfig{
		Image:       appImage,
		Verbose:     false,
		Memory:      512,
		Networking:  "nat",
		NatRules:    []nat.Rule{nat.Rule{GuestPort: "10000", HostPort: "10000"}},
		BackingFile: false,
		Cmd:         osvCmdline,
		DisableKvm:  r.DisableKvm,
		AioType:     r.QemuAioType,
	}

	// TODO Have to come up with a better error handling if necessary. Be more verbose on errors.
	cmd, err := qemu.VMCommand(vmconfig, verbose)
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Finally, let's start the command: launch the VM
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Make sure the process is always properly killed, even in case of unhandled exception
	defer cmd.Process.Kill()

	go io.Copy(os.Stderr, stderr)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		if verbose {
			fmt.Println(text)
		}

		// We are looking for the following message from the OSv guest.
		if text == "Waiting for connection from host..." {
			// Cancel the scanner as soon as this message has been received.
			break
		}
	}

	// Consuming stdout is mandatory once it is redirected to linux socket.
	// If not, buffer will fill up and capstan will hang.
	go io.Copy(ioutil.Discard, stdout)

	conn, err := util.ConnectAndWait("tcp", "localhost:10000")
	if err != nil {
		if !r.DisableKvm && strings.Contains(err.Error(), "getsockopt: connection refused") {
			// Probably KVM is already in use e.g. by VirtualBox. Suggest user to turn it off for qemu.
			fmt.Println("Could not run QEMU VM. Try to set 'disable_kvm:true' in ~/.capstan/config.yaml")
		}
		return nil, err
	}
	defer conn.Close()

	newHashes, err := uploadFiles(conn, uploadPaths, imageCache, verbose)
	if err != nil {
		return nil, err
	} else {
		return newHashes, cmd.Wait()
	}
}

func UploadPackageContentsToRemoteGuest(uploadPaths map[string]string, remoteHostNameOrIpAddress string, verbose bool) error {

	fmt.Printf("Uploading files to %s...\n", remoteHostNameOrIpAddress)

	conn, err := util.ConnectAndWait("tcp", remoteHostNameOrIpAddress+":10000")
	if err != nil {
		if strings.Contains(err.Error(), "getsockopt: connection refused") {
			fmt.Println("Could not connect to " + remoteHostNameOrIpAddress)
		}
		return err
	}
	defer conn.Close()

	_, err = uploadFiles(conn, uploadPaths, nil, verbose)
	return err
}

func uploadFiles(conn net.Conn, uploadPaths map[string]string, imageCache core.HashCache, verbose bool) (core.HashCache, error) {
	// Initialise a progress bar for uploading files. Only start it in case
	// silent mode is activated.
	var bar *pb.ProgressBar
	if !verbose {
		tmpl := `{{ yellow "Uploading files " }} {{ bar . "<" "-" (cycle . "↖" "↗" "↘" "↙" ) "." ">"}} {{speed . | green}} {{percent . | blue}}`
		bar = pb.ProgressBarTemplate(tmpl).Start(len(uploadPaths))
	}

	// Initialise a dictionary for the up-to-date file hashes.
	newHashes := core.NewHashCache()

	// Loop over collected paths and upload them to the image if necessary.
	for src, dest := range uploadPaths {
		// Get the hash of this path.
		hash, _ := hashPath(src, dest)

		// By default it should upload all files, except those whose cached
		// hash value hasn't changed since the last upload.
		uploadFile := true
		if cachedHash, ok := imageCache[dest]; ok {
			// If hashes are the same, we should not upload.
			uploadFile = hash != cachedHash
		}

		if uploadFile {
			// Upload the file from host to guest. This will access cpiod
			// running in OSv.
			err := CopyFile(conn, src, dest)
			if err != nil {
				return nil, err
			}

			if verbose {
				fmt.Printf("Adding %s  --> %s \n", src, dest)
			}
		} else if verbose {
			fmt.Printf("Skipping %s  --> %s\n", src, dest)
		}

		if !verbose {
			bar.Increment()
		}

		// Store the new hash whenever a file is successfully uploaded to the VM.
		newHashes[dest] = hash
	}

	if !verbose {
		bar.Finish()
	}

	// Finalise the transfer.
	cpio.WritePadded(conn, cpio.ToWireFormat("TRAILER!!!", 0, 0))

	return newHashes, nil
}

func CollectPathContents(path string) (map[string]string, error) {
	fi, err := os.Stat(path)

	// Check that path exists.
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist", path)
	}

	// Make sure that the upload path is absolute. This will also make sure that any trailing slashes
	// are properly handled.
	path, err = filepath.Abs(path)

	contents := make(map[string]string)

	switch {
	case fi.Mode().IsDir():
		// Look into the upload folder and add all files from there to the list.
		err = filepath.Walk(path, func(p string, info os.FileInfo, _ error) error {
			if p != path {
				contents[p] = strings.TrimPrefix(p, path)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}

	case fi.Mode().IsRegular():
		contents[path] = "/" + filepath.Base(path)
	}

	return contents, nil
}

func hashPath(hostPath, vmPath string) (string, error) {
	info, err := os.Stat(hostPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("Unable to hash unexistent path: %s", hostPath)
	}

	var data []byte
	switch {
	case info.IsDir():
		data = []byte(vmPath)
	default:
		data, err = ioutil.ReadFile(hostPath)
		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", md5.Sum(data)), nil
}
