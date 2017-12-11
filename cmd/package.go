/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package cmd

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
	"time"

	"github.com/mikelangelo-project/capstan/core"
	"github.com/mikelangelo-project/capstan/runtime"
	"github.com/mikelangelo-project/capstan/util"
	"gopkg.in/yaml.v2"
)

func InitPackage(packagePath string, p *core.Package) error {
	// Remember when the package was initialized.
	p.Created = core.YamlTime{time.Now()}

	// We have to create the package directory and it's metadata directory.
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

	gzWriter := gzip.NewWriter(mpmfile)
	defer gzWriter.Close()
	tarball := tar.NewWriter(gzWriter)
	defer tarball.Close()

	err = filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(path, packageDir)

		// TODO(miha-plesko): respect .capstanignore instead hard-coding

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
func ComposePackage(repo *util.Repo, imageSize int64, updatePackage, verbose, pullMissing bool,
	packageDir, appName string, bootOpts *BootOptions) error {

	// Package content should be collected in a subdirectory called mpm-pkg.
	targetPath := filepath.Join(packageDir, "mpm-pkg")
	// Remove collected directory afterwards.
	defer os.RemoveAll(targetPath)

	// Construct final bootcmd for the image.
	commandLine, err := bootOpts.GetCmd()
	if err != nil {
		return err
	}

	// First, collect the contents of the package.
	if err := CollectPackage(repo, packageDir, pullMissing, false, verbose); err != nil {
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

func ComposePackageAndUploadToRemoteInstance(repo *util.Repo, verbose, pullMissing bool, packageDir, remoteHostInstance string) error {

	// Package content should be collected in a subdirectory called mpm-pkg.
	targetPath := filepath.Join(packageDir, "mpm-pkg")
	// Remove collected directory afterwards.
	defer os.RemoveAll(targetPath)

	// First, collect the contents of the package.
	if err := CollectPackage(repo, packageDir, pullMissing, true, verbose); err != nil {
		return err
	}

	// If all is well, we have to start preparing the files for upload.
	paths, err := collectDirectoryContents(targetPath)
	if err != nil {
		return err
	}

	return UploadPackageContentsToRemoteGuest(paths, remoteHostInstance, verbose)
}

// CollectPackage will try to resolve all of the dependencies of the given package
// and collect the content in the $CWD/mpm-pkg directory.
func CollectPackage(repo *util.Repo, packageDir string, pullMissing, remote, verbose bool) error {
	// Get the manifest file of the given package.
	pkg, err := core.ParsePackageManifest(filepath.Join(packageDir, "meta", "package.yaml"))
	if err != nil {
		return err
	}

	genRuntime, err := runtime.PackageRunManifestGeneral(filepath.Join(packageDir, "meta", "run.yaml"))
	if err != nil {
		return err
	}

	// If runtime is known, then we add runtime dependencies to the list.
	if genRuntime != nil && len(genRuntime.GetDependencies()) > 0 {
		fmt.Printf("Prepending '%s' runtime dependencies to dep list: %s\n",
			genRuntime.GetRuntimeName(), genRuntime.GetDependencies())
		pkg.Require = append(genRuntime.GetDependencies(), pkg.Require...)
	}

	// The bootstrap package is implicitly required by every application package,
	// so we add it to the list of required packages. Even if user has added
	// the bootstrap manually, this will not result in overhead. There is only
	// one exception to this: when ComposeRemote is invoked, the bootstrap package
	// mustn't be required since it clashes with executables that are already in the
	// remote unikernel.
	if remote {
		pkg.Require = append([]string{"osv.compose-remote"}, pkg.Require...)
	} else {
		pkg.Require = append([]string{"osv.bootstrap"}, pkg.Require...)
	}

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

	allCmdConfigs := &runtime.AllCmdConfigs{}

	// First collect everything from the required packages.
	for _, req := range requiredPackages {
		reader, err := repo.GetPackageTarReader(req.Name)
		if err != nil {
			return err
		}

		cmdConf, err := extractPackageContent(reader, targetPath, req.Name)
		if err != nil {
			return err
		}
		allCmdConfigs.Add(req.Name, cmdConf)
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
			cmdConf, err := runtime.ParsePackageRunManifestData(data)
			if err != nil {
				return err
			}
			allCmdConfigs.Add(pkg.Name, cmdConf)
			return nil
		} else if relPath == "/meta" { // Prevent empty `/meta` dir from being uploaded.
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

	// Persist all boot commands into /run directory.
	if err := allCmdConfigs.Persist(targetPath); err != nil {
		return err
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

func extractPackageContent(tarReader *tar.Reader, target, pkgName string) (*runtime.CmdConfig, error) {
	var cmdConf *runtime.CmdConfig
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				// Have we reached till the end of the tar?
				break
			}
			return nil, err
		}

		if absTarPathMatches(header.Name, "/meta/run.yaml") {
			// Prepare files with boot commands for this package.
			data, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, err
			}
			cmdConf, err = runtime.ParsePackageRunManifestData(data)
			if err != nil {
				return nil, err
			}
			continue
		} else if absTarPathMatches(header.Name, "/meta/.*") {
			// Skip other manifest data
			continue
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()

		switch {
		case info.Mode()&os.ModeSymlink == os.ModeSymlink:
			if err := ensureDirectoryStructureForFile(path); err != nil {
				return nil, fmt.Errorf("Could not prepare directory structure for %s: %s", path, err)
			}

			// Create symbolic link. Ignore any error that might occur locally as
			// links can be created dynamically on the VM itself.
			os.Symlink(header.Linkname, path)

		case info.IsDir():
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return nil, err
			}

		case info.Mode().IsRegular():
			if err := ensureDirectoryStructureForFile(path); err != nil {
				return nil, fmt.Errorf("Could not prepare directory structure for %s: %s", path, err)
			}

			writer, err := os.Create(path)
			if err != nil {
				return nil, err
			}

			_, err = io.Copy(writer, tarReader)
			err = os.Chmod(path, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}

			writer.Close()

		default:
			return nil, fmt.Errorf("File %s has unsupported mode %v", path, info.Mode())
		}
	}

	return cmdConf, nil
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
func DescribePackage(repo *util.Repo, packageName string) (string, error) {
	if !repo.PackageExists(packageName) {
		return "", fmt.Errorf("Package %s does not exist in your local repository. Pull it using "+
			"'capstan package pull %s'", packageName, packageName)
	}

	tarReader, err := repo.GetPackageTarReader(packageName)
	if err != nil {
		return "", err
	}

	var pkg *core.Package
	var cmdConf *runtime.CmdConfig
	var readme string

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				// Have we reached till the end of the tar?
				break
			}
			return "", err
		}

		if absTarPathMatches(header.Name, "/meta/package.yaml") {
			data, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return "", err
			}
			pkg = &core.Package{}
			if err := pkg.Parse(data); err != nil {
				return "", err
			}
		} else if absTarPathMatches(header.Name, "/meta/run.yaml") {
			data, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return "", err
			}
			if cmdConf, err = runtime.ParsePackageRunManifestData(data); err != nil {
				return "", err
			}
		} else if absTarPathMatches(header.Name, "/meta/README.md") {
			data, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return "", err
			}
			readme = string(data)
		}

		// Stop reading if we have all the information
		if pkg != nil && cmdConf != nil && readme != "" {
			break
		}
	}

	s := fmt.Sprintln("PACKAGE METADATA")
	if pkg != nil {
		s += fmt.Sprintln("name:", pkg.Name)
		s += fmt.Sprintln("title:", pkg.Title)
		s += fmt.Sprintln("author:", pkg.Author)

		if len(pkg.Require) > 0 {
			s += fmt.Sprintln("required packages:")
			for _, r := range pkg.Require {
				s += fmt.Sprintf("   * %s\n", r)
			}
		}
	} else {
		return "", fmt.Errorf("package is not valid: missing meta/package.yaml")
	}

	s += fmt.Sprintln()

	if cmdConf != nil {
		s += fmt.Sprintln("PACKAGE EXECUTION")
		s += fmt.Sprintln("runtime:", cmdConf.RuntimeType)
		if cmdConf.ConfigSetDefault == "" && len(cmdConf.ConfigSets) == 1 {
			for configName := range cmdConf.ConfigSets {
				s += fmt.Sprintln("default configuration:", configName)
			}
		} else {
			s += fmt.Sprintln("default configuration:", cmdConf.ConfigSetDefault)
		}

		s += fmt.Sprintln("-----------------------------------------")
		s += fmt.Sprintf("%-25s | %s\n", "CONFIGURATION NAME", "BOOT COMMAND")
		s += fmt.Sprintln("-----------------------------------------")
		for configName := range cmdConf.ConfigSets {
			bootCmd, err := cmdConf.ConfigSets[configName].GetBootCmd(nil, nil)
			if err != nil {
				return "", err
			}
			s += fmt.Sprintf("%-25s | %s\n", configName, bootCmd)
		}
		s += fmt.Sprintln("-----------------------------------------")
	} else {
		s += fmt.Sprintln("No package execution information was found.")
	}

	s += fmt.Sprintln("")

	if readme != "" {
		s += fmt.Sprintln("PACKAGE DOCUMENTATION")
		s += fmt.Sprintln(readme)
	}

	return s, nil
}

type BootOptions struct {
	Cmd        string
	Boot       []string
	EnvList    []string
	PackageDir string
}

// GetCmd builds final bootcmd based on three parameters (in this order):
// * --run <commandLine>
// * --boot <customBoot>
// * config_set_default: <> (read from meta/run.yaml within packageDir)
func (b *BootOptions) GetCmd() (string, error) {
	command := ""

	if b.Cmd != "" { // Direct commandLine has highest priority (--run <commandLine>).
		fmt.Println("Command line will be set based on --run parameter")
		command = b.Cmd
	} else if len(b.Boot) > 0 { // Configuration name has second-highest priority (--boot <customBoot>).
		fmt.Println("Command line will be set based on --boot parameters")
		command = runtime.BootCmdForScript(b.Boot)
	} else if b.PackageDir != "" { // Default configuration in yaml has third-highest priority (config_set_default: <>).
		if data, err := ioutil.ReadFile(filepath.Join(b.PackageDir, "meta", "run.yaml")); err == nil {
			if cmdConf, err := runtime.ParsePackageRunManifestData(data); err == nil && cmdConf.ConfigSetDefault != "" {
				fmt.Println("Command line will be set based on config_set_default attribute of meta/run.yaml")
				command = runtime.BootCmdForScript(strings.Split(cmdConf.ConfigSetDefault, ","))
			}
		}
	} else { // Fallback is empty bootcmd.
		fmt.Println("Empty command line will be set for this image")
		command = ""
	}

	// Prepend environment variables to the command.
	if env, err := util.ParseEnvironmentList(b.EnvList); err == nil {
		if command, err = runtime.PrependEnvsPrefix(command, env, false); err != nil {
			return "", err
		}
	} else {
		return "", err
	}

	return command, nil
}

// absTarPathMatches tells whether the tar header name matches the path pattern.
// This function is needed since some tar files prefix its header names
// with / and some not. NOTE: 'pathPattern' is always considered absolute path
// regardles if it starts with / or not. 'tarPath' parameter should be header.Name.
func absTarPathMatches(tarPath, pathPattern string) (res bool) {
	pathPattern = strings.TrimPrefix(pathPattern, "/")
	res, _ = regexp.MatchString(fmt.Sprintf("^/?%s$", pathPattern), tarPath)
	return
}
