package lint

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Example line to match:
// ./parse.go:9:9: main redeclared in this block
var lintParagraph = regexp.MustCompile(`^([^: ]+):([0-9]+)(?::([0-9]+):)? (.*)$`)

const metadataPrefix = "# githublinter: "

type Paragraph struct {
	File string
	Line uint32
	Char uint32
	Msg  []string
}

func (m *Paragraph) Message() string {
	return strings.Join(m.Msg, "\n")
}

func (m *Paragraph) String() string {
	return fmt.Sprintf("%s:%d:%d: %s", m.File, m.Line, m.Char, m.Message())
}

func Parse(reader io.Reader) ([]*Paragraph, error) {
	paragraphs := []*Paragraph{}

	scanner := bufio.NewScanner(reader)

	var currentMatch *Paragraph
	var metadata string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, metadataPrefix) {
			metadata = strings.TrimPrefix(line, metadataPrefix)
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		groups := lintParagraph.FindAllStringSubmatch(line, -1)
		if groups == nil && currentMatch != nil {
			currentMatch.Msg = append(currentMatch.Msg, line)
		} else if groups != nil {
			line, err := strconv.ParseInt(groups[0][2], 10, 32)
			if err != nil {
				return []*Paragraph{}, err
			}
			char, err := strconv.ParseInt(groups[0][2], 10, 32)
			if err != nil {
				return []*Paragraph{}, err
			}

			var msg []string
			if metadata != "" {
				msg = []string{fmt.Sprintf("%s: %s", metadata, groups[0][4])}
			} else {
				msg = []string{groups[0][4]}
			}
			currentMatch = &Paragraph{
				File: groups[0][1],
				Line: uint32(line),
				Char: uint32(char),
				Msg:  msg,
			}
			paragraphs = append(paragraphs, currentMatch)
		}

	}

	if err := scanner.Err(); err != nil {
		return []*Paragraph{}, err
	}

	return paragraphs, nil
}
