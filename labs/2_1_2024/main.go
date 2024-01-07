package main

import (
	"fmt"
	"os"

	labs "github.com/FeurJak/Prudence/labs"

	"github.com/urfave/cli/v2"
)

/*
	Lab 2.1.2024:

	Test EVM Integration

	- Prepare EVM with specified state
	- Get ERC20 Token Meta

	run: go build && ./2_1_2024 run "connection string" "token address" "base token address" "factory address" "router address"
	i.e. go build && ./2_1_2024 run geth.ipc 0x1673AB963C825402596F63e3d1Ef2c2966aa5340 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2 0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f 0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D

	i.e connection string --> rpc URL or IPC Path

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
