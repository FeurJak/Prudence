package main

import (
	"fmt"
	"os"

	labs "github.com/Prudence/labs"

	"github.com/urfave/cli/v2"
)

/*
	Lab 2.1.2024:

	Test EVM Integration

	- Prepare EVM with specified state
	- Get ERC20 Token Meta

	run: go build && ./2_1_2024 run "connection string", i.e IPC Path

	Context:
	Although the P2P stack is independent, precog still relies on state access.
	State access can be done in two ways:
		1. Accessing state from disk
		2. Accessing state from a remote geth node

	This lab uses remote state access, which means that precog will connect to a geth node
	However, make sure that your node is running in Archive Node


*/

var lab = labs.NewLab("lab_2_1_2024")

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
