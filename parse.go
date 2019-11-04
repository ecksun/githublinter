package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/docopt/docopt-go"
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

// Example line to match:
// ./parse.go:9:9: main redeclared in this block
var lintParagraph = regexp.MustCompile(`^([^: ]+):([0-9]+)(?::([0-9]+):)? (.*)$`)

type paragraph struct {
	file string
	line uint32
	char uint32
	msg  []string
}

func (m *paragraph) Message() string {
	return strings.Join(m.msg, "\n")
}

func (m *paragraph) String() string {
	return fmt.Sprintf("%s:%d:%d: %s", m.file, m.line, m.char, m.Message())
}

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
}`, c.path, c.position, c.body)
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
	paragraphs, err := parseLint(reader)
	if err != nil {
		panic(err)
	}

	fileLints := map[string][]*paragraph{}
	for _, issue := range paragraphs {
		file := path.Clean(issue.file)
		if _, exists := fileLints[file]; !exists {
			fileLints[file] = []*paragraph{}
		}
		fileLints[file] = append(fileLints[file], issue)
	}

	comments := []graphQLComment{}

	for _, file := range diff.Files {
		if lints, exists := fileLints[path.Clean(file.NewName)]; exists {
			diffLines := getNewLinesInDiff(file)
			// This is O(log(n)*m) where n = #diffLines, m = #lints
			for _, lint := range lints {
				position, ok := getDiffPosition(diffLines, int(lint.line)) // TODO Fix int types
				if !ok {
					fmt.Fprintf(os.Stderr, "%s:%d not found in diff\n", lint.file, lint.line)
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

func parseLint(reader io.Reader) ([]*paragraph, error) {
	paragraphs := []*paragraph{}

	scanner := bufio.NewScanner(reader)

	var currentMatch *paragraph

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		groups := lintParagraph.FindAllStringSubmatch(line, -1)
		if groups == nil && currentMatch != nil {
			currentMatch.msg = append(currentMatch.msg, line)
		} else if groups != nil {
			line, err := strconv.ParseInt(groups[0][2], 10, 32)
			if err != nil {
				return []*paragraph{}, err
			}
			char, err := strconv.ParseInt(groups[0][2], 10, 32)
			if err != nil {
				return []*paragraph{}, err
			}

			currentMatch = &paragraph{
				file: groups[0][1],
				line: uint32(line),
				char: uint32(char),
				msg:  []string{groups[0][4]},
			}
			paragraphs = append(paragraphs, currentMatch)
		}

	}

	if err := scanner.Err(); err != nil {
		return []*paragraph{}, err
	}

	return paragraphs, nil
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
