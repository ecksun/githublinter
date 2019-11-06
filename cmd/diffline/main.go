package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/docopt/docopt-go"
	"github.com/waigani/diffparser"
)

func main() {
	usage := `diffline

Usage:
  diffline <diffile> <filename> <line>
  diffline -h | --help
  diffline --version

Options:
  -h --help     Show this screen.
  --version     Show version.`

	var conf struct {
		Diffile  string
		Filename string
		Line     int
	}

	arguments, err := docopt.ParseDoc(usage)
	arguments.Bind(&conf)

	bytes, err := ioutil.ReadFile(conf.Diffile)
	diff, err := diffparser.Parse(string(bytes))

	if err != nil {
		panic(err)
	}

	for _, file := range diff.Files {
		fmt.Printf("%s == %s\n", file.NewName, conf.Filename)
		if file.NewName == conf.Filename {
			lines := getNewLinesInDiff(file)
			printDiffLines(lines)
			position, ok := getDiffPosition(lines, conf.Line)
			if !ok {
				fmt.Fprintf(os.Stderr, "Line %d not found in diff\n", conf.Line)
				os.Exit(1)
			}
			fmt.Println(position)
			os.Exit(0)
		}
	}
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

func getDiffPosition(newLines []*diffparser.DiffLine, lineNumber int) (position int, found bool) {
	i := sort.Search(len(newLines), func(i int) bool { return newLines[i].Number >= lineNumber })
	if i < len(newLines) && newLines[i].Number == lineNumber {
		return newLines[i].Position, true
	}
	return -1, false
}

func printDiffLines(lines []*diffparser.DiffLine) {
	for _, line := range lines {
		fmt.Printf("%+v\n", line)
	}
}
