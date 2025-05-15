package main

import (
	cli "Blockchain_Test/cli"
	// wallet "Blockchain_Test/wallet"
)

func main() {
	cmd := cli.CommandLine{}
	cmd.Run()

	// w := wallet.MakeWallet()
	// w.Address()
}
