package main

import (
	"fmt"

	"github.com/ecksun/diffline/pkg/github"
)

func main() {
	pr, err := github.GetPR("ecksun", "test-repo", 3)

	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", pr)
}
