package main

import (
	"fmt"
	"os"

	"github.com/grassrootseconomics/go-vise/testdata"
)

func main() {
	var err error
	if len(os.Args) > 1 {
		err = testdata.GenerateTo(os.Args[1])
	} else {
		_, err = testdata.Generate()
	}
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(testdata.DataDir)
}
