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
