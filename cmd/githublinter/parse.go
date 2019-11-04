package main

import (
	"fmt"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/ecksun/diffline/pkg/difflint"
)

const usage string = `githublinter

Usage:
  githublinter <diffile> <lintfile>
  githublinter -h | --help
  githublinter --version

Options:
  -h --help     Show this screen.
  --version     Show version.`

func main() {
	var conf struct {
		Diffile  string
		Lintfile string
	}

	arguments, err := docopt.ParseDoc(usage)
	arguments.Bind(&conf)

	diffReader, err := os.Open(conf.Diffile)
	if err != nil {
		panic(err)
	}

	lintReader, err := os.Open(conf.Lintfile)
	if err != nil {
		panic(err)
	}

	comments, err := difflint.GetLintIssuesInDiff(diffReader, lintReader)
	if err != nil {
		panic(err)
	}

	fmt.Print("[")
	for i, comment := range comments {
		fmt.Print(comment)
		if i != len(comments)-1 {
			fmt.Print(", ")
		}
	}
	fmt.Println("]")
}
