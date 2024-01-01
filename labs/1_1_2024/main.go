package main

import (
	"fmt"
	"os"

	labs "github.com/Prudence/labs"

	"github.com/urfave/cli/v2"
)

/*
	Lab 1.1.2024:

	Test P2P Network

	- Connect To Peers
	- Get Mempool % Coverage over time
*/

var lab = labs.NewLab("lab_1_1_2024")

func init() {
	lab.Commands = []*cli.Command{runCommand}
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
