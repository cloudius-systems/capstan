/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package capstan

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type Repo struct {
	Path string
}

func NewRepo() *Repo {
	return &Repo{
		Path: filepath.Join(os.Getenv("HOME"), "/.capstan/repository/"),
	}
}

func (r *Repo) PullImage(image string) error {
	if r.ImageExists(image) {
		return errors.New(fmt.Sprintf("%s: image already exists", image))
	}
	fmt.Printf("Pulling %s...\n", image)
	gitUrl := fmt.Sprintf("https://github.com/%s", image)
	cmd := exec.Command("git", "clone", gitUrl, filepath.Join(r.Path, image))
	_, err := cmd.Output()
	if err != nil {
		return errors.New(fmt.Sprintf("%s: no such remote image", image))
	}
	return nil
}

func (r *Repo) PushImage(image string) {
	fmt.Printf("Pushing %s...\n", image)
	cmd := exec.Command("cp", image, r.Path)
	out, err := cmd.Output()
	if err != nil {
		println(err.Error())
		return
	}
	print(string(out))
}

func (r *Repo) ImageExists(image string) bool {
	file := r.ImagePath(image)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

func (r *Repo) RemoveImage(image string) {
	if !r.ImageExists(image) {
		fmt.Printf("%s: no such image\n", image)
		return
	}
	file := r.ImagePath(image)
	fmt.Printf("Removing %s...\n", image)
	cmd := exec.Command("rm", "-rf", filepath.Dir(file))
	out, err := cmd.Output()
	if err != nil {
		println(err.Error())
		return
	}
	print(string(out))
}

func (r *Repo) ImagePath(image string) string {
	return filepath.Join(r.Path, image, filepath.Base(image))
}

func (r *Repo) ListImages() {
	namespaces, _ := ioutil.ReadDir(r.Path)
	for _, n := range namespaces {
		images, _ := ioutil.ReadDir(filepath.Join(r.Path, n.Name()))
		nrImages := 0
		for _, i := range images {
			if i.IsDir() {
				fmt.Println(n.Name() + "/" + i.Name())
				nrImages++
			}
		}
		// Image is directly at repository root with no namespace:
		if nrImages == 0 && n.IsDir() {
			fmt.Println(n.Name())
		}
	}
}
