package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/ecksun/diffline/pkg/lint"
	"github.com/waigani/diffparser"
)

const usage string = `githublinter

Usage:
  githublinter <diffile> <lintfile>
  githublinter -h | --help
  githublinter --version

Options:
  -h --help     Show this screen.
  --version     Show version.`

type graphQLComment struct {
	path     string
	position int
	body     string
}

func (c graphQLComment) String() string {
	return fmt.Sprintf(`{
    path: "%s"
    position: %d
    body: "%s"
}`, c.path, c.position, strings.ReplaceAll(c.body, "\"", "\\\""))
}

func main() {
	var conf struct {
		Diffile  string
		Lintfile string
	}

	arguments, err := docopt.ParseDoc(usage)
	arguments.Bind(&conf)

	bytes, err := ioutil.ReadFile(conf.Diffile)
	diff, err := diffparser.Parse(string(bytes))

	if err != nil {
		panic(err)
	}

	reader, err := os.Open(conf.Lintfile)
	paragraphs, err := lint.Parse(reader)
	if err != nil {
		panic(err)
	}

	fileLints := map[string][]*lint.Paragraph{}
	for _, issue := range paragraphs {
		file := path.Clean(issue.File)
		if _, exists := fileLints[file]; !exists {
			fileLints[file] = []*lint.Paragraph{}
		}
		fileLints[file] = append(fileLints[file], issue)
	}

	comments := []graphQLComment{}

	for _, file := range diff.Files {
		if lints, exists := fileLints[path.Clean(file.NewName)]; exists {
			diffLines := getNewLinesInDiff(file)
			// This is O(log(n)*m) where n = #diffLines, m = #lints
			for _, lint := range lints {
				position, ok := getDiffPosition(diffLines, int(lint.Line)) // TODO Fix int types
				if !ok {
					fmt.Fprintf(os.Stderr, "%s:%d not found in diff\n", lint.File, lint.Line)
				}
				comments = append(comments, graphQLComment{
					path:     file.NewName,
					position: position,
					body:     lint.Message(),
				})
			}
		}
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

func getNewLinesInDiff(file *diffparser.DiffFile) []*diffparser.DiffLine {
	var lines []*diffparser.DiffLine
	for _, hunk := range file.Hunks {
		for _, line := range hunk.NewRange.Lines {
			lines = append(lines, line)
		}
	}

	return lines
}

// O(log(lines))
func getDiffPosition(lines []*diffparser.DiffLine, lineNumber int) (position int, found bool) {
	i := sort.Search(len(lines), func(i int) bool { return lines[i].Number >= lineNumber })
	if i < len(lines) && lines[i].Number == lineNumber {
		return lines[i].Position, true
	}
	return -1, false
}
