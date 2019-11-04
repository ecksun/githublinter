package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/ecksun/diffline/pkg/difflint"
	"github.com/ecksun/diffline/pkg/github"
)

// Matches for example https://github.com/ecksun/test-repo/pull/3
var githubURLRegex = regexp.MustCompile(`.*//github.com/([^/]+)/([^/]+)/pull/([0-9]+)`)

func main() {
	groups := githubURLRegex.FindAllStringSubmatch("https://github.com/ecksun/test-repo/pull/3", -1)
	if groups == nil {
		panic("Failed to parse url")
	}
	owner := groups[0][1]
	repo := groups[0][2]
	pr := groups[0][3]

	pullrequest, err := github.GetPR(owner, repo, pr)
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("git", "diff", pullrequest.BaseRef, pullrequest.HeadRef)
	cmd.Dir = os.Getenv("GIT_REPO")

	if cmd.Dir == "" {
		panic("Empty GIT_REPO")
	}

	// var diffBuffer bytes.Buffer
	// cmd.Stdout = &diffBuffer

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	if err = cmd.Start(); err != nil {
		panic(err)
	}

	go func() {
		err := cmd.Wait()
		if err != nil {
			panic(err)
		}
	}()

	repoDir := os.Getenv("GIT_REPO")

	lints := []*exec.Cmd{
		exec.Command("go", "vet"),
	}

	lintOut, err := lints[0].StderrPipe()
	if err != nil {
		panic(err)
	}

	for _, lint := range lints {
		lint.Dir = repoDir
		if err := lint.Start(); err != nil {
			panic(err)
		}
	}

	for _, lint := range lints {
		go func() {
			err := lint.Wait()
			if err != nil {
				fmt.Println(err)
			}
		}()
	}

	comments, err := difflint.GetLintIssuesInDiff(stdout, lintOut)
	if err != nil {
		panic(err)
	}

	fmt.Println(comments)
}
