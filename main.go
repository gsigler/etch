package main

import (
	"fmt"
	"os"

	"github.com/gsigler/etch/cmd"
	etcherr "github.com/gsigler/etch/internal/errors"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			etcherr.Render(etcherr.New(etcherr.CatIO, fmt.Sprintf("unexpected panic: %v", r)), true)
			os.Exit(2)
		}
	}()

	verbose, err := cmd.Execute()
	if err != nil {
		etcherr.Render(err, verbose)
		os.Exit(1)
	}
}
