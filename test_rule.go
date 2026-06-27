package main

import (
	"fmt"
	"github.com/pocketbase/pocketbase/tools/search"
)

func main() {
	rules := []string{
		"dismissed_by.id !?= @request.auth.id",
		"dismissed_by.id ?!= @request.auth.id",
		"dismissed_by.id != @request.auth.id",
		"@request.auth.id !~ dismissed_by.id",
		"dismissed_by.id !~ @request.auth.id",
	}

	for _, r := range rules {
		_, err := search.FilterData(r).BuildExpr(search.NewSimpleFieldResolver("id", "dismissed_by.id", "@request.auth.id"))
		if err != nil {
			fmt.Printf("Rule %q INVALID: %v\n", r, err)
		} else {
			fmt.Printf("Rule %q VALID\n", r)
		}
	}
}
