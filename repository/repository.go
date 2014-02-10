/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package repository

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func PullImage(image string) {
	fmt.Printf("Pulling %s...\n", image)
	gitUrl := fmt.Sprintf("https://github.com/%s", image)
	cmd := exec.Command("git", "clone", gitUrl, filepath.Join(RepoPath(), image))
	out, err := cmd.Output()
	if err != nil {
		println(err.Error())
		return
	}
	print(string(out))
}

func PushImage(image string) {
	fmt.Printf("Pushing %s...\n", image)
	repo := RepoPath()
	cmd := exec.Command("cp", image, repo)
	out, err := cmd.Output()
	if err != nil {
		println(err.Error())
		return
	}
	print(string(out))
}

func RemoveImage(image string) {
	file := ImagePath(image)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		fmt.Printf("%s: no such image\n", image)
		return
	}
	fmt.Printf("Removing %s...\n", image)
	cmd := exec.Command("rm", "-rf", filepath.Dir(file))
	out, err := cmd.Output()
	if err != nil {
		println(err.Error())
		return
	}
	print(string(out))
}

func RepoPath() string {
	return filepath.Join(os.Getenv("HOME"), "/.capstan/repository/")
}

func ImagePath(image string) string {
	return filepath.Join(RepoPath(), image, filepath.Base(image))
}

func ListImages() {
	repo := RepoPath()
	namespaces, _ := ioutil.ReadDir(repo)
	for _, n := range namespaces {
		images, _ := ioutil.ReadDir(filepath.Join(repo, n.Name()))
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
