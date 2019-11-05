package common

import (
	"fmt"
	"strings"
)

type GraphQLComment struct {
	Path     string
	Position int
	Body     string
}

func (c GraphQLComment) String() string {
	return fmt.Sprintf(`{
    path: "%s"
    position: %d
    body: "%s"
}`, c.Path, c.Position, strings.ReplaceAll(c.Body, "\"", "\\\""))
}
