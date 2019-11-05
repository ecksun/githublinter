package difflint

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"

	"github.com/ecksun/diffline/pkg/common"
	"github.com/ecksun/diffline/pkg/lint"
	"github.com/waigani/diffparser"
)

func GetLintIssuesInDiff(rawDiff io.Reader, rawLints io.Reader) ([]common.GraphQLComment, error) {
	bytes, err := ioutil.ReadAll(rawDiff)
	if err != nil {
		return nil, err
	}

	diff, err := diffparser.Parse(string(bytes))

	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %v", err)
	}

	paragraphs, err := lint.Parse(rawLints)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lint output: %v", err)
	}

	fileLints := map[string][]*lint.Paragraph{}
	for _, issue := range paragraphs {
		file := path.Clean(issue.File)
		if _, exists := fileLints[file]; !exists {
			fileLints[file] = []*lint.Paragraph{}
		}
		fileLints[file] = append(fileLints[file], issue)
	}

	comments := []common.GraphQLComment{}

	for _, file := range diff.Files {
		if lints, exists := fileLints[path.Clean(file.NewName)]; exists {
			diffLines := getNewLinesInDiff(file)
			// This is O(log(n)*m) where n = #diffLines, m = #lints
			for _, lint := range lints {
				position, ok := getDiffPosition(diffLines, int(lint.Line)) // TODO Fix int types
				if !ok {
					fmt.Fprintf(os.Stderr, "%s:%d not found in diff\n", lint.File, lint.Line)
				}
				comments = append(comments, common.GraphQLComment{
					Path:     file.NewName,
					Position: position,
					Body:     lint.Message(),
				})
			}
		}
	}

	return comments, nil
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
