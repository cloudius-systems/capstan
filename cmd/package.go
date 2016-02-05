package cmd

import (
	"archive/tar"
	"fmt"
	"github.com/cloudius-systems/capstan/core"
	"github.com/cloudius-systems/capstan/util"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = relPath

		if err := tarball.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		defer file.Close()
		_, err = io.Copy(tarball, file)
		return err
	})

	if err != nil {
		return "", err
	}

	fmt.Printf("Package built and stored in %s\n", target)

	return target, nil
}

func ComposePackage(repo *util.Repo, imageSize int64, packageDir string, appName string) error {
	targetPath := filepath.Join(packageDir, "mpm-pkg")
	// Remove collected directory afterwards.
	defer os.Remove(targetPath)

	// First, collect the contents of the package.
	err := CollectPackage(repo, packageDir)
	if err != nil {
		return err
	}

	// If all is well, we have to start preparing the files for upload.
	paths, err := collectDirectoryContents(targetPath)
	if err != nil {
		return err
	}

	// Initialize an empty image based on the provided loader image. imageSize is used to
	// determine the size of the user partition. Use default loader image.
	if err := repo.InitializeImage("", appName, imageSize); err != nil {
		return fmt.Errorf("Failed to initialize empty image named %s", appName)
	}

	// Get the path of imported image.
	imagePath := repo.ImagePath("qemu", appName)

	// Upload the specified path onto virtual image.
	if err := UploadPackageContents(imagePath, paths); err != nil {
		return err
	}

	return nil
}

func CollectPackage(repo *util.Repo, packageDir string) error {
	// Get the manifest file of the given package.
	pkg, err := core.ParsePackageManifest(filepath.Join(packageDir, "meta", "package.yaml"))
	if err != nil {
		return err
	}

	// Look for all dependencies and make sure they are all available in the repository.
	requiredPackages, err := repo.GetPackageDependencies(pkg)
	if err != nil {
		return err
	}

	targetPath := filepath.Join(packageDir, "mpm-pkg")
	if err = os.MkdirAll(targetPath, 0775); err != nil {
		return err
	}

	err = filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(path, packageDir)
		if relPath != "" && !strings.HasPrefix(relPath, "/meta") && !strings.HasPrefix(relPath, "/mpm-pkg") {
			// If the current path is a directory, create target directory and skip to the next path.
			if info.IsDir() {
				if err = os.MkdirAll(filepath.Join(targetPath, relPath), info.Mode()); err != nil {
					return err
				}

				return nil
			}

			err = util.CopyLocalFile(filepath.Join(targetPath, relPath), path)
			if err != nil {
				return err
			}
		}

		return nil
	})

	for _, req := range requiredPackages {
		reqpkg, err := repo.GetPackage(req.Name)
		if err != nil {
			return err
		}

		err = extractPackageContent(reqpkg, targetPath)
		if err != nil {
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

func extractPackageContent(pkgreader io.Reader, target string) error {
	tarReader := tar.NewReader(pkgreader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// Have we reached till the end of the tar?
			break
		} else if err != nil {
			return err
		}

		// Skip manifest data.
		if strings.HasPrefix(header.Name, "/meta") {
			continue
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}

			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}

		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}

	return nil
}
