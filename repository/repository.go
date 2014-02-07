/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package repository

import (
	"fmt"
	"os"
	"os/exec"
)

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
	cmd := exec.Command("rm", file)
	out, err := cmd.Output()
	if err != nil {
		println(err.Error())
		return
	}
	print(string(out))
}

func RepoPath() string {
	home := os.Getenv("HOME")
	return home + "/.capstan/repository/"
}

func ImagePath(image string) string {
	return RepoPath() + "/" + image
}

func ListImages() {
	repo := RepoPath()
	cmd := exec.Command("ls", "-1", repo)
	out, err := cmd.Output()
	if err != nil {
		println(err.Error())
		return
	}
	print(string(out))
}
