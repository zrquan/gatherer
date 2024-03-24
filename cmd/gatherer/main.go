package main

import (
	"fmt"
	"os"

	"github.com/zrquan/gatherer/pkg/core"
)

const version = "0.1.0"

func main() {
	fmt.Printf("API Caver v%s\n\n", version)

	opts, err := core.ParseOptions()
	if err != nil {
		fmt.Println("Parse options error:", err)
		os.Exit(125)
	}

	runner, err := core.NewRunner(opts)
	if err != nil {
		fmt.Println("Build runner error:", err)
		os.Exit(125)
	}

	runner.Execute()
}
