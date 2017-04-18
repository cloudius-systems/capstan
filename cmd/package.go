/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mikelangelo-project/capstan/core"
	"github.com/mikelangelo-project/capstan/runtime"
	"github.com/mikelangelo-project/capstan/util"
	"gopkg.in/yaml.v2"
)

func InitPackage(packagePath string, p *core.Package) error {
	// We have to create hte package directory and it's metadata directory.
	metaPath := filepath.Join(packagePath, "meta")

	fmt.Printf("Initializing package in %s\n", metaPath)

	// Create the meta dir.
	if err := os.MkdirAll(metaPath, 0755); err != nil {
		return err
	}

	// Save basic package data to YAML.
	d, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	// Save serialised YAML file to the appropriate place in the metadata directory.
	err = ioutil.WriteFile(filepath.Join(metaPath, "package.yaml"), d, 0644)
	if err != nil {
		return err
	}

	return nil
}

func BuildPackage(packageDir string) (string, error) {
	fmt.Println("Building package")

	pkg, err := core.ParsePackageManifest(filepath.Join(packageDir, "meta", "package.yaml"))
	if err != nil {
		return "", err
	}

	mpmname := fmt.Sprintf("%s.mpm", pkg.Name)
	target := filepath.Join(packageDir, mpmname)
	mpmfile, err := os.Create(target)
	if err != nil {
		return "", err
	}

	defer mpmfile.Close()

	tarball := tar.NewWriter(mpmfile)
	defer tarball.Close()

	err = filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(path, packageDir)

		// Skip the MPM package file or the collected package content..
		if filepath.Base(path) == mpmname || strings.HasPrefix(relPath, "/mpm-pkg") {
			return nil
		}

		link := ""
		// Check whether the current path is a link
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			// Get the link target. It is relative to the link.
			//if link, err = filepath.EvalSymlinks(path); err != nil {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}

		// Link is an empty string in case the path represents a regular file or dir.
		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}

		// Since the default initialisation uses only the basename for the name
		// we have to use a path relative to the package in order to presserve
		// hierarchy.
		if info.IsDir() {
			header.Name = relPath + "/"
		} else {
			header.Name = relPath
		}

		if err := tarball.WriteHeader(header); err != nil {
			return err
		}

		switch {
		case info.Mode()&os.ModeSymlink == os.ModeSymlink:
			return nil

		case info.Mode().IsDir():
			return nil

		case info.Mode().IsRegular():
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			defer file.Close()
			_, err = io.Copy(tarball, file)

			return err

		default:
			return fmt.Errorf("File %s has unsupported mode %v", path, info.Mode())
		}
	})

	if err != nil {
		return "", err
	}

	fmt.Printf("Package built and stored in %s\n", target)

	return target, nil
}

// ComposePackage uses the contents of the specified package directory and
// create a (QEMU) virtual machine image. The image consists of all of the
// required packages.
// If updatePackage is set, ComposePackage tries to update an existing image
// by comparing previous MD5 hashes to the ones in the current package
// directory. Only modified files are uploaded and no file deletions are
// possible at this time.
func ComposePackage(repo *util.Repo, imageSize int64, updatePackage bool, verbose bool,
	pullMissing bool, customBoot string, packageDir string, appName string, commandLine string) error {

	// Package content should be collected in a subdirectory called mpm-pkg.
	targetPath := filepath.Join(packageDir, "mpm-pkg")
	// Remove collected directory afterwards.
	defer os.RemoveAll(targetPath)

	// Construct final bootcmd for the image.
	commandLine = constructBootCmdFromArguments(commandLine, customBoot, packageDir)

	// First, collect the contents of the package.
	if err := CollectPackage(repo, packageDir, pullMissing, customBoot, verbose); err != nil {
		return err
	}

	// If all is well, we have to start preparing the files for upload.
	paths, err := collectDirectoryContents(targetPath)
	if err != nil {
		return err
	}

	// Get the path of imported image.
	imagePath := repo.ImagePath("qemu", appName)
	// Check whether the image already exists.
	imageExists := false
	if _, err = os.Stat(imagePath); !os.IsNotExist(err) {
		imageExists = true
	}

	imageCachePath := repo.ImageCachePath("qemu", appName)
	var imageCache core.HashCache

	// If the user requested new image or requested to update a non-existent image,
	// initialize it first.
	if !updatePackage || !imageExists {
		// Initialize an empty image based on the provided loader image. imageSize is used to
		// determine the size of the user partition. Use default loader image.
		if err := repo.InitializeImage("", appName, imageSize); err != nil {
			return fmt.Errorf("Failed to initialize empty image named %s.\nError was: %s", appName, err)
		}
	} else {
		// We are updating an existing image so try to parse the cache
		// config file. Note that we are not interested in any errors as
		// no-cache or invalid cache means that all files will be uploaded.
		imageCache, _ = core.ParseHashCache(imageCachePath)
	}

	// Upload the specified path onto virtual image.
	imageCache, err = UploadPackageContents(repo, imagePath, paths, imageCache, verbose)
	if err != nil {
		return err
	}

	// Save the new image cache
	imageCache.WriteToFile(imageCachePath)

	// Set the command line.
	if err = util.SetCmdLine(imagePath, commandLine); err != nil {
		return err
	}
	fmt.Printf("Command line set to: '%s'\n", commandLine)

	return nil
}

// CollectPackage will try to resolve all of the dependencies of the given package
// and collect the content in the $CWD/mpm-pkg directory.
func CollectPackage(repo *util.Repo, packageDir string, pullMissing bool, customBoot string, verbose bool) error {
	// Get the manifest file of the given package.
	pkg, err := core.ParsePackageManifest(filepath.Join(packageDir, "meta", "package.yaml"))
	if err != nil {
		return err
	}

	runtime, err := core.PackageRunManifestGeneral(filepath.Join(packageDir, "meta", "run.yaml"))
	if err != nil {
		return err
	}

	// If runtime is known, then we add runtime dependencies to the list.
	if runtime != nil && len(runtime.GetDependencies()) > 0 {
		fmt.Printf("Prepending '%s' runtime dependencies to dep list: %s\n",
			runtime.GetRuntimeName(), runtime.GetDependencies())
		pkg.Require = append(runtime.GetDependencies(), pkg.Require...)
	}

	// The bootstrap package is implicitly required by every application package,
	// so we add it to the list of required packages. Even if user has added
	// the bootstrap manually, this will not result in overhead.
	pkg.Require = append(pkg.Require, "eu.mikelangelo-project.osv.bootstrap")

	// Look for all dependencies and make sure they are all available in the repository.
	requiredPackages, err := repo.GetPackageDependencies(pkg, pullMissing)
	if err != nil {
		return err
	}

	targetPath := filepath.Join(packageDir, "mpm-pkg")

	// Delete old 'mpm-package' folder if exists
	if _, err := os.Stat(targetPath); err == nil {
		if err = os.RemoveAll(targetPath); err != nil {
			fmt.Printf("failed to remove 'mpm-pkg' folder: %s\n", err)
		}
	}

	if err = os.MkdirAll(targetPath, 0775); err != nil {
		return err
	}

	// First collect everything from the required packages.
	for _, req := range requiredPackages {
		reqpkg, err := repo.GetPackage(req.Name)
		if err != nil {
			return err
		}

		err = extractPackageContent(reqpkg, targetPath, req.Name)
		if err != nil {
			return err
		}
	}

	// Read .capstanignore if exists.
	capstanignorePath := filepath.Join(packageDir, ".capstanignore")
	if _, err := os.Stat(capstanignorePath); os.IsNotExist(err) {
		if verbose {
			fmt.Println("WARN: .capstanignore not found, all files will be uploaded")
		}
		capstanignorePath = ""
	}
	capstanignore, err := core.CapstanignoreInit(capstanignorePath)
	if err != nil {
		return err
	}

	// Now we need to append the content of the current package into the target directory.
	// This should override any file from the required packages.
	err = filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		link := ""
		// Check whether the current path is a link
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			// Get the link target. It is relative to the link.
			//if link, err = filepath.EvalSymlinks(path); err != nil {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}

		relPath := strings.TrimPrefix(path, packageDir)

		// Apply meta/run.yaml before ignoring it.
		if relPath == "/meta/run.yaml" {
			// Prepare files with boot commands.
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			if err := persistBootCmdsIntoFiles(data, targetPath, customBoot, ""); err != nil {
				return err
			}
			return nil
		}

		// Ignore what needs to be ignored.
		if capstanignore.IsIgnored(relPath) {
			if verbose {
				suffix := ""
				if info.IsDir() {
					suffix = " (entire folder)"
				}
				fmt.Printf(".capstanignore: ignore %s%s\n", relPath, suffix)
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		switch {
		case info.Mode()&os.ModeSymlink == os.ModeSymlink:
			return os.Symlink(link, filepath.Join(targetPath, relPath))

		case info.IsDir():
			return os.MkdirAll(filepath.Join(targetPath, relPath), info.Mode())

		case info.Mode().IsRegular():
			return util.CopyLocalFile(filepath.Join(targetPath, relPath), path)

		default:
			return fmt.Errorf("File %s has unsupported mode %v", path, info.Mode())
		}
	})

	if err != nil {
		return err
	}

	if runtime != nil {
		if err := runtime.OnCollect(targetPath); err != nil {
			return err
		}
	}

	return nil
}

func collectDirectoryContents(packageDir string) (map[string]string, error) {
	packageDir, err := filepath.Abs(packageDir)

	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s does not exist", packageDir)
	}

	contents := make(map[string]string)

	err = filepath.Walk(packageDir, func(path string, info os.FileInfo, _ error) error {
		relPath := strings.TrimPrefix(path, packageDir)
		// Ignore package's meta data
		if relPath != "" && !strings.HasPrefix(relPath, "/meta") {
			contents[path] = relPath
		}
		return nil
	})

	return contents, err
}

func ImportPackage(repo *util.Repo, packageDir string) error {
	packagePath, err := BuildPackage(packageDir)
	if err != nil {
		return err
	}

	pkg, err := core.ParsePackageManifest(filepath.Join(packageDir, "meta", "package.yaml"))
	if err != nil {
		return err
	}

	defer os.Remove(packagePath)

	// Import the package into the current repository.
	return repo.ImportPackage(pkg, packagePath)
}

func extractPackageContent(pkgreader io.Reader, target, pkgName string) error {
	tarReader := tar.NewReader(pkgreader)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				// Have we reached till the end of the tar?
				break
			}
			return err
		}

		if header.Name == "meta/run.yaml" {
			// Prepare files with boot commands for this package.
			data, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return err
			}
			if err := persistBootCmdsIntoFiles(data, target, "", pkgName); err != nil {
				return err
			}
			continue
		} else if strings.HasPrefix(header.Name, "/meta/") || strings.HasPrefix(header.Name, "meta/") {
			// Sometimes path starts with slash and sometimes not. Best to catch both here.

			// Skip other manifest data
			continue
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()

		switch {
		case info.Mode()&os.ModeSymlink == os.ModeSymlink:
			if err := ensureDirectoryStructureForFile(path); err != nil {
				return fmt.Errorf("Could not prepare directory structure for %s: %s", path, err)
			}

			// Create symbolic link. Ignore any error that might occur locally as
			// links can be created dynamically on the VM itself.
			os.Symlink(header.Linkname, path)

		case info.IsDir():
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}

		case info.Mode().IsRegular():
			if err := ensureDirectoryStructureForFile(path); err != nil {
				return fmt.Errorf("Could not prepare directory structure for %s: %s", path, err)
			}

			writer, err := os.Create(path)
			if err != nil {
				return err
			}

			_, err = io.Copy(writer, tarReader)
			err = os.Chmod(path, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			writer.Close()

		default:
			return fmt.Errorf("File %s has unsupported mode %v", path, info.Mode())
		}
	}

	return nil
}

// PullPackage looks for the package in remote repository and tries to import
// it into local repository.
func PullPackage(r *util.Repo, packageName string) error {
	// Try to download the package from the remote repository.
	return r.DownloadPackage(r.URL, packageName)
}

// ensureDirectoryStructureForFile creates directory path for given filepath.
func ensureDirectoryStructureForFile(currfilepath string) error {
	dirpath := filepath.Dir(currfilepath)

	if _, err := os.Stat(dirpath); err != nil {
		if err = os.MkdirAll(dirpath, 0775); err != nil {
			return err
		}
	}

	return nil
}

// DescribePackage describes package with given name without extracting it.
func DescribePackage(repo *util.Repo, packageName string) error {
	if !repo.PackageExists(packageName) {
		return fmt.Errorf("Package %s does not exist in your local repository. Pull it using "+
			"'capstan package pull %s'", packageName, packageName)
	}

	pkgTar, err := repo.GetPackage(packageName)
	if err != nil {
		return err
	}

	var pkg *core.Package
	var cmdConf *core.CmdConfig

	tarReader := tar.NewReader(pkgTar)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				// Have we reached till the end of the tar?
				break
			}
			return err
		}

		// Read meta/package.yaml
		if strings.HasSuffix(header.Name, "meta/package.yaml") {
			data, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return err
			}
			pkg = &core.Package{}
			if err := pkg.Parse(data); err != nil {
				return err
			}
		}

		// Read meta/run.yaml
		if strings.HasSuffix(header.Name, "meta/run.yaml") {
			data, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return err
			}
			if cmdConf, err = core.ParsePackageRunManifestData(data); err != nil {
				return err
			}
		}

		// Stop reading if we have all the information
		if pkg != nil && cmdConf != nil {
			break
		}
	}

	fmt.Println("PACKAGE METADATA")
	if pkg != nil {
		fmt.Println("name:", pkg.Name)
		fmt.Println("title:", pkg.Title)
		fmt.Println("author:", pkg.Author)

		if len(pkg.Require) > 0 {
			fmt.Println("required packages:")
			for _, r := range pkg.Require {
				fmt.Printf("   * %s\n", r)
			}
		}
	} else {
		return fmt.Errorf("package is not valid: missing meta/package.yaml")
	}

	fmt.Println("")

	if cmdConf != nil {
		fmt.Println("PACKAGE EXECUTION")
		fmt.Println("runtime:", cmdConf.RuntimeType)
		if cmdConf.ConfigSetDefault == "" && len(cmdConf.ConfigSets) == 1 {
			for configName := range cmdConf.ConfigSets {
				fmt.Println("default configuration:", configName)
			}
		} else {
			fmt.Println("default configuration:", cmdConf.ConfigSetDefault)
		}

		fmt.Println("-----------------------------------------")
		fmt.Printf("%-25s | %s\n", "CONFIGURATION NAME", "BOOT COMMAND")
		fmt.Println("-----------------------------------------")
		for configName := range cmdConf.ConfigSets {
			bootCmd, err := cmdConf.ConfigSets[configName].GetBootCmd()
			if err != nil {
				return err
			}
			fmt.Printf("%-25s | %s\n", configName, bootCmd)
		}
		fmt.Println("-----------------------------------------")
	} else {
		fmt.Println("No package execution information was found.")
	}

	return nil
}

// persistBootCmdsIntoFiles iterates configuration sets and generates bootcmd file for each.
// These files can then be used by OSv bootloader to run thread based on --boot parameter.
// Argument mpmFolder should point to the root of the OSv i.e. mpm-pkg folder. Prefix is used to
// prefix 'default' configuration filename. E.g. prefix "abc" results in filename /run/abc-default.
func persistBootCmdsIntoFiles(runYamlData []byte, mpmFolder, customBoot string, prefix string) error {
	cmdConf, err := core.ParsePackageRunManifestData(runYamlData)
	if err != nil {
		return err
	}

	// Prepare folder to store bootcmd files in.
	targetFolder := filepath.Join(mpmFolder, "run")
	if _, err := os.Stat(targetFolder); err != nil {
		if err = os.MkdirAll(targetFolder, 0775); err != nil {
			return err
		}
	}

	// Calculate bootcmd for each config set and persist it to file.
	for confName := range cmdConf.ConfigSets {
		currConf := cmdConf.ConfigSets[confName]

		// Validate.
		if err := currConf.Validate(); err != nil {
			return fmt.Errorf("Validation failed for configuration set '%s': %s", confName, err)
		}

		// Calculate boot command.
		bootCmd, err := currConf.GetBootCmd()
		if err != nil {
			return err
		}

		// Persist to file.
		cmdFile := filepath.Join(targetFolder, confName)
		if err := ioutil.WriteFile(cmdFile, []byte(bootCmd), 0775); err != nil {
			return err
		}
	}

	// Argument --boot <name> has greater priority than config_set_default in meta/run.yaml
	if customBoot != "" {
		cmdConf.ConfigSetDefault = customBoot
	}

	// TODO: Add symbolic links to point to default configset of each package.
	// Use name 'default' for this package's link and '{prefix}-default' for other.

	return nil
}

// constructBootCmdFromArguments builds bootcmd based on three parameters (in this order):
// * --run <commandLine>
// * --boot <customBoot>
// * config_set_default: <> (read from meta/run.yaml within packageDir)
func constructBootCmdFromArguments(commandLine, customBoot, packageDir string) string {
	// Direct commandLine has highest priority (--run <commandLine>).
	if commandLine != "" {
		fmt.Println("Command line will be set based on --run parameter")
		return commandLine
	}

	// Configuration name has second-highest priority (--boot <customBoot>).
	if customBoot != "" {
		fmt.Println("Command line will be set based on --boot parameter")
		return runtime.BootCmdForScript(customBoot)
	}

	// Default configuration in yaml has third-highest priority (config_set_default: <>).
	if data, err := ioutil.ReadFile(filepath.Join(packageDir, "meta", "run.yaml")); err == nil {
		if cmdConf, err := core.ParsePackageRunManifestData(data); err == nil && cmdConf.ConfigSetDefault != "" {
			fmt.Println("Command line will be set based on config_set_default attribute of meta/run.yaml")
			return runtime.BootCmdForScript(cmdConf.ConfigSetDefault)
		}
	}

	// Fallback is empty bootcmd.
	fmt.Println("Empty command line will be set for this image")
	return ""
}
