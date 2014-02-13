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
	root := os.Getenv("CAPSTAN_ROOT")
	if root == "" {
		root = filepath.Join(os.Getenv("HOME"), "/.capstan/repository/") 
	}
	return &Repo{
		Path: root,
	}
}

func (r *Repo) PullImage(image string) error {
	if r.ImageExists(image) {
		return r.updateImage(image)
	}
	fmt.Printf("Pulling %s...\n", image)
	gitUrl := fmt.Sprintf("https://github.com/%s", image)
	workTree := r.workTree(image)
	cmd := exec.Command("git", "clone", "--depth", "1", gitUrl, workTree)
	_, err := cmd.Output()
	if err != nil {
		return errors.New(fmt.Sprintf("%s: unable to pull remote image", image))
	}
	return nil
}

func (r *Repo) updateImage(image string) error {
	fmt.Printf("Updating %s...\n", image)
	workTree := r.workTree(image)
	gitDir   := r.gitDir(image)
	cmd := exec.Command("git", "--git-dir", gitDir, "--work-tree", workTree, "remote", "update")
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	cmd = exec.Command("git", "--git-dir", gitDir, "--work-tree", workTree, "merge", "origin/master")
	_, err = cmd.Output()
	return err
}

func (r *Repo) gitDir(image string) string {
	return filepath.Join(r.workTree(image), ".git")
}

func (r *Repo) workTree(image string) string {
	return filepath.Join(r.Path, image)
}

func (r *Repo) PushImage(image string, file string) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("%s: no such file", file))
	}
	fmt.Printf("Pushing %s...\n", image)
	cmd := exec.Command("mkdir", "-p", filepath.Dir(r.ImagePath(image)))
	_, err := cmd.Output()
	if err != nil {
		return errors.New(fmt.Sprintf("%s: mkdir failed", filepath.Dir(r.ImagePath(image))))
	}
	cmd = exec.Command("cp", file, r.ImagePath(image))
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	return nil
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
