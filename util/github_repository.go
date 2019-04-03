/*
 * Copyright (C) 2014 Cloudius Systems, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package util

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (r *Repo) PullImage(image string) error {
	workTree := r.workTree(image)
	_, err := os.Stat(workTree)
	if os.IsNotExist(err) {
		return r.cloneImage(image)
	}
	if err != nil {
		return err
	}
	return r.updateImage(image)
}

func (r *Repo) cloneImage(image string) error {
	fmt.Printf("Pulling %s...\n", image)
	workTree := r.workTree(image)
	gitUrl := fmt.Sprintf("https://github.com/%s", image)
	cmd := exec.Command("git", "clone", "--depth", "1", gitUrl, workTree)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return errors.New(fmt.Sprintf("%s: unable to pull remote image", image))
	}
	return nil
}

func (r *Repo) updateImage(image string) error {
	fmt.Printf("Updating %s...\n", image)
	workTree := r.workTree(image)
	gitDir := r.gitDir(image)
	cmd := exec.Command("git", "--git-dir", gitDir, "--work-tree", workTree, "remote", "update")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return err
	}
	cmd = exec.Command("git", "--git-dir", gitDir, "--work-tree", workTree, "merge", "origin/master")
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return err
	}
	return nil
}

func (r *Repo) gitDir(image string) string {
	return filepath.Join(r.workTree(image), ".git")
}

func (r *Repo) workTree(image string) string {
	return filepath.Join(r.RepoPath(), image)
}
