package cmd

import (
	"bufio"
	"fmt"
	"github.com/cloudius-systems/capstan/cpio"
	"github.com/cloudius-systems/capstan/hypervisor/qemu"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/util"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Compose(r *util.Repo, loaderImage string, imageSize int64, uploadPath string, appName string) error {
	// Initialize an empty image based on the provided loader image. imageSize is used to
	// determine the size of the user partition.
	err := r.InitializeImage(loaderImage, appName, imageSize)

	// Get the path of imported image.
	imagePath := r.ImagePath("qemu", appName)

	paths, err := CollectPathContents(uploadPath)
	if err != nil {
		return err
	}

	// Upload the specified path onto virtual image.
	if err = UploadPackageContents(imagePath, paths); err != nil {
		return err
	}

	return nil
}

func UploadPackageContents(appImage string, uploadPaths map[string]string) error {
	// Specify the VM properties. Use the app image as the source to start.
	vmconfig := &qemu.VMConfig{
		Image:       appImage,
		Verbose:     false,
		Memory:      512,
		Networking:  "nat",
		NatRules:    []nat.Rule{nat.Rule{GuestPort: "10000", HostPort: "10000"}},
		BackingFile: false,
		// It is asumed that the UploadPath is the first command executed by this virtual image. Thus
		// we also create the filesystem and start the 'cpiod' daemon responsible for copying files
		// to target VM.
		Cmd: "--norandom --nomount --noinit /tools/mkfs.so; /tools/cpiod.so --prefix /zfs/zfs; /zfs.so set compression=off osv",
	}

	// TODO Have to come up with a better error handling if necessary. Be more verbose on errors.
	cmd, err := qemu.VMCommand(vmconfig)
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Finally, let's start the command: launch the VM
	if err := cmd.Start(); err != nil {
		return err
	}

	// Make sure the process is always properly killed, even in case of unhandled exception
	defer cmd.Process.Kill()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		// We are looking for the following message from the OSv guest.
		if text == "Waiting for connection from host..." {
			// Cancel the scanner as soon as this message has been received.
			break
		}
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	conn, err := util.ConnectAndWait("tcp", "localhost:10000")
	defer conn.Close()
	if err != nil {
		return err
	}

	for src, dest := range uploadPaths {
		err = CopyFile(conn, src, dest)
		if err != nil {
			return err
		}
	}

	cpio.WritePadded(conn, cpio.ToWireFormat("TRAILER!!!", 0, 0))

	return cmd.Wait()
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
