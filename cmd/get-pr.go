package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/ecksun/diffline/pkg/difflint"
	"github.com/ecksun/diffline/pkg/github"
)

const usage string = `githublinter

Usage:
  githublinter <pr> <linter>...
  githublinter -h | --help
  githublinter --version

Options:
  -h --help     Show this screen.
  --version     Show version.`

// Matches for example https://github.com/ecksun/test-repo/pull/3
var githubURLRegex = regexp.MustCompile(`.*//github.com/([^/]+)/([^/]+)/pull/([0-9]+)`)

func main() {
	var conf struct {
		PR     string `docopt:"<pr>"`
		Linter []string
	}

	arguments, err := docopt.ParseDoc(usage)
	if err != nil {
		panic(err)
	}

	arguments.Bind(&conf)

	groups := githubURLRegex.FindAllStringSubmatch(conf.PR, -1)
	if groups == nil {
		fmt.Fprintf(os.Stderr, "Invalid github pull-request URL: %s\n", conf.PR)
		os.Exit(2)
	}

	owner := groups[0][1]
	repo := groups[0][2]
	pr := groups[0][3]

	repoDir := os.Getenv("GIT_REPO")
	if repoDir == "" {
		panic("Empty GIT_REPO")
	}

	pullrequest, err := github.GetPR(owner, repo, pr)
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("git", "diff", pullrequest.BaseRef, pullrequest.HeadRef)
	cmd.Dir = os.Getenv("GIT_REPO")

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

	lintPath, err := getLintPath(conf.Linter[0])
	if err != nil {
		panic(err)
	}

	lintCmd := exec.Command(lintPath, conf.Linter[1:]...)

	lintOut, err := lintCmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	lintCmd.Dir = repoDir

	if err := lintCmd.Start(); err != nil {
		panic(err)
	}

	go func() {
		err := lintCmd.Wait()
		if err != nil {
			fmt.Println(err)
		}
	}()

	comments, err := difflint.GetLintIssuesInDiff(stdout, lintOut)
	if err != nil {
		panic(err)
	}

	fmt.Println(comments)
}

func getLintPath(execPath string) (string, error) {
	if strings.Contains(execPath, "/") {
		return filepath.Abs(execPath)
	}
	return execPath, nil
}