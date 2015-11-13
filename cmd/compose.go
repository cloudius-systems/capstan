package cmd

import (
	"bufio"
	"fmt"
	"github.com/cloudius-systems/capstan/cpio"
	"github.com/cloudius-systems/capstan/hypervisor/qemu"
	"github.com/cloudius-systems/capstan/nat"
	"github.com/cloudius-systems/capstan/nbd"
	"github.com/cloudius-systems/capstan/util"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func Compose(r *util.Repo, loaderImage string, imageSize int64, uploadPath string, appName string) error {
	loaderImagePath := r.ImagePath("raw", loaderImage)
	// Check whether the base launcher image exists
	loaderInfo, err := os.Stat(loaderImagePath)
	if os.IsNotExist(err) {
		fmt.Println("The specified loader image (%s) does not exist.", loaderImagePath)
		return err
	}

	// Create temporary folder in which the image will be composed.
	tmp, _ := ioutil.TempDir("", "capstan")
	imagePath := path.Join(tmp, "application.img")

	// Copy the OSv base iamge into application image
	if err := util.CopyLocalFile(imagePath, loaderImagePath); err != nil {
		return err
	}

	// Get the size of the loader image, then round that to the closest 2MB to start the user
	// ZFS partition.
	zfsStart := (loaderInfo.Size() + 2097151) & ^2097151
	// Make filesystem size in bytes
	zfsSize := int64(imageSize * 1024 * 1024)

	// Make sure the image is in QCOW2 format. This is to make sure that the
	// image in the next step does not grow in size in case the input image is
	// in RAW format.
	if err := nbd.SetPartition(imagePath, 2, uint64(zfsStart), uint64(zfsSize)); err != nil {
		fmt.Printf("Setting the ZFS partition failed for %s\n", imagePath)
		return err
	}

	// Now that the partition has been created, resize the virtual image size.
	if err := util.ConvertImageToQCOW2(imagePath); err != nil {
		return err
	}

	// Now that the partition has been created, resize the virtual image size.
	if err := util.ResizeImage(imagePath, uint64(zfsSize+zfsStart)); err != nil {
		fmt.Printf("Failed to set the target size (%db) of the image %s\n", (zfsSize + zfsStart), imagePath)
		return err
	}

	// Set the command to initialize the ZFS partition created above and start listening for CPIOD requests.
	if err := nbd.SetCmdLine(imagePath, "--norandom --nomount --noinit /tools/mkfs.so; /tools/cpiod.so --prefix /zfs/zfs; /zfs.so set compression=off osv"); err != nil {
		fmt.Printf("Setting the command line to initialize the VM failed")
		return err
	}

	if err = UploadPath(imagePath, uploadPath); err != nil {
		return err
	}

	// The image is now composed and we can move it into the repository.
	r.ImportImage(appName, imagePath, "", time.Now().Format(time.RFC3339), "", "")

	if err = os.RemoveAll(tmp); err != nil {
		fmt.Println(err)
	}

	return nil
}

func UploadPath(appImage string, uploadPath string) error {
	// Make sure that the upload path is absolute. This will also make sure that any trailing slashes
	// are properly handled.
	uploadPath, err := filepath.Abs(uploadPath)

	fi, err := os.Stat(uploadPath)

	if os.IsNotExist(err) {
		fmt.Printf("The given path (%s) does not exist\n", uploadPath)
		return err
	}

	// Specify the VM properties. Use the app image as the source to start.
	vmconfig := &qemu.VMConfig{
		Image:       appImage,
		Verbose:     false,
		Memory:      512,
		Networking:  "nat",
		NatRules:    []nat.Rule{nat.Rule{GuestPort: "10000", HostPort: "10000"}},
		BackingFile: false,
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
	if err != nil {
		return err
	}

	var pathPrefix string
	files := make([]string, 0)
	// Check whether the upload path is file or folder.
	switch {
	case fi.Mode().IsDir():
		pathPrefix = uploadPath
		// Look into the upload folder and add all files from there to the list.
		err = filepath.Walk(uploadPath, func(path string, info os.FileInfo, _ error) error {
			if path != uploadPath {
				files = append(files, path)
			}

			return nil
		})

	case fi.Mode().IsRegular():
		pathPrefix = path.Dir(uploadPath)
		// If it is just the file, add it to the list alone.
		files = append(files, uploadPath)
	}

	for _, file := range files {
		destFile := strings.TrimPrefix(file, pathPrefix)
		//fmt.Printf("%s --> %s\n", file, destFile)
		err = CopyFile(conn, file, destFile)
	}

	cpio.WritePadded(conn, cpio.ToWireFormat("TRAILER!!!", 0, 0))

	conn.Close()
	return cmd.Wait()
}
