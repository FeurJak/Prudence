package main

import (
	"fmt"
	"os"

	labs "github.com/FeurJak/Prudence/labs"

	"github.com/urfave/cli/v2"
)

/*
	Lab 2.1.2024:

	ERC20 Token Transfer


*/

var lab = labs.NewLab("lab_8_1_2024")

func init() {
	lab.Commands = []*cli.Command{runCommand, accCommand}
}

func main() {
	if err := lab.Run(os.Args); err != nil {
		code := 1
		if ec, ok := err.(*labs.NumberedError); ok {
			code = ec.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}

}
