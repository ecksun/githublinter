package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/ecksun/diffline/pkg/common"
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

const prMessage = `
# Linting issues found!

githublinter found linting issues introduced in this pull-request.
`

const updatePRMessage = prMessage + `
New linting issues were found after the original review, they are in detailed in a review below.
`

func main() {
	var conf struct {
		PR     string `docopt:"<pr>"`
		Linter []string
	}

	arguments, err := docopt.ParseDoc(usage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse document options, this is likely a programmer error: %+v\n", err)
		os.Exit(2)
	}

	arguments.Bind(&conf)

	groups := githubURLRegex.FindAllStringSubmatch(conf.PR, -1)
	if groups == nil {
		fmt.Fprintf(os.Stderr, "Invalid github pull-request URL: %s\n", conf.PR)
		os.Exit(1)
	}

	owner := groups[0][1]
	repo := groups[0][2]
	pr := groups[0][3]

	repoDir := os.Getenv("GIT_REPO")
	if repoDir == "" {
		fmt.Fprintf(os.Stderr, "You must set the GIT_REPO environent variable to an up-to-date clone of the repository\n")
		os.Exit(1)
	}

	pullrequest, err := github.GetPR(owner, repo, pr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get pull-request from github: %+v\n", err)
		os.Exit(1)
	}

	existingComments := map[common.GraphQLComment]struct{}{}

	for _, review := range pullrequest.Reviews {
		for _, comment := range review.Comments {
			graphQLComment := common.GraphQLComment{
				Path:     comment.Path,
				Position: comment.Position,
				Body:     comment.Body,
			}
			existingComments[graphQLComment] = struct{}{}
		}
	}

	comments, err := getLinterComments(pullrequest, repoDir, conf.Linter[0], conf.Linter[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get and parse linter output: %+v\n", err)
		os.Exit(1)
	}

	var newComments []common.GraphQLComment
	for _, comment := range comments {
		if _, exists := existingComments[comment]; !exists {
			newComments = append(newComments, comment)
		}
	}

	if len(newComments) != 0 {
		if len(pullrequest.Reviews) == 0 {
			err := github.CreateReview(pullrequest.ID, prMessage, newComments)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create pull-request review: %+v\n", err)
				os.Exit(1)
			}
			fmt.Println("Successfully submitted review")

		} else {
			err := github.UpdateReview(pullrequest.Reviews[0].ID, updatePRMessage)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to update pull-request body: %+v\n", err)
				os.Exit(1)
			}
			fmt.Println("Successfully updated pull-request review body")
			err = github.CreateReview(pullrequest.ID, "", newComments)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create pull-request review: %+v\n", err)
				os.Exit(1)
			}
			fmt.Println("Successfully added new pull-request review with additional comments")

		}
	} else {
		fmt.Println("No issues found!")
	}
}

func getLinterComments(pullrequest github.PullRequest, repoDir string, linter string, linterArgs []string) ([]common.GraphQLComment, error) {
	if pullrequest.BaseRef == "" || pullrequest.HeadRef == "" {
		return nil, fmt.Errorf("Cannot get diff between %q and %q", pullrequest.BaseRef, pullrequest.HeadRef)
	}
	cmd := exec.Command("git", "diff", pullrequest.BaseRef, pullrequest.HeadRef)
	cmd.Dir = os.Getenv("GIT_REPO")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout of git diff: %+v", err)
	}

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start git diff: %+v", err)
	}

	lintPath, err := getLintPath(linter)
	if err != nil {
		return nil, fmt.Errorf("failed to get path to linter: %+v", err)
	}

	lintCmd := exec.Command(lintPath, linterArgs...)

	lintOut, err := lintCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout of linter command: %+v", err)
	}

	lintCmd.Dir = repoDir

	if err := lintCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start linter: %+v", err)
	}

	comments, err := difflint.GetLintIssuesInDiff(stdout, lintOut)

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("git diff exited with a non-zero exit status: %+v", err)
	}

	// We ignore errors as linting will fail if there are lint-errors,
	// which is just fine in our case
	_ = lintCmd.Wait()

	return comments, err
}

func getLintPath(execPath string) (string, error) {
	if strings.Contains(execPath, "/") {
		return filepath.Abs(execPath)
	}
	return execPath, nil
}
