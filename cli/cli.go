package cli

import (
	blockchain "Blockchain_Test/Blockchain"
	wallet "Blockchain_Test/wallet"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"bufio"
)

type CommandLine struct{
	Consistency string
    NodeID      string
    TrustScore  float64
	NodeManagementMode bool
}

func (cli *CommandLine) printUsage() {
    fmt.Println("Usage:")
    fmt.Println(" getbalance -address ADDRESS - Get balance of ADDRESS")
    fmt.Println(" createwallet - Create a new wallet")
    fmt.Println(" listaddresses - List all addresses in the wallet")
    fmt.Println(" createblockchain -address ADDRESS - Create a new blockchain")
    fmt.Println(" printchain - Print the blocks in the blockchain")
    fmt.Println(" send -from FROM -to TO -amount AMOUNT - Send amount")
    fmt.Println(" validatechain - Validate the blockchain integrity")
    fmt.Println(" setconsistency [strong|eventual] - Set consistency mode")
    fmt.Println(" settrust -node NODE_ID -score SCORE - Set trust score")
}

func (cli *CommandLine) printChain() {
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Timestamp: %d\n", block.Timestamp)
		fmt.Printf("State Root: %x\n", block.StateRoot)
		fmt.Printf("Entropy Value: %.4f\n", block.EntropyValue)

		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createBlockChain(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	chain := blockchain.InitBlockChain(address)
	chain.Database.Close()
	fmt.Println("Blockchain created successfully!")
}

func (cli *CommandLine) getBalance(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	chain := blockchain.ContinueBlockChain(address)
	defer chain.Database.Close()

	balance := 0
	UTXOs := chain.FindUTXO(address)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int) {
	if !wallet.ValidateAddress(from) {
		log.Panic("From address is not valid")
	}
	if !wallet.ValidateAddress(to) {
		log.Panic("To address is not valid")
	}
	chain := blockchain.ContinueBlockChain(from)
	defer chain.Database.Close()

	tx := blockchain.NewTransaction(from, to, amount, chain)
	chain.AddBlock([]*blockchain.Transaction{tx})
	fmt.Println("Transaction successful!")
}

func (cli *CommandLine) createWallet() {
	wallets, _ := wallet.CreateWallets()
	address := wallets.AddWallet()
	wallets.SaveFile()

	fmt.Printf("New address: %s\n", address)
}

func (cli *CommandLine) listAddresses() {
	wallets, _ := wallet.CreateWallets()
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) validateChain() {
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()
	
	iter := chain.Iterator()
	isValid := true
	
	for {
		block := iter.Next()
		
		if !chain.VerifyBlockIntegrity(block) {
			isValid = false
			fmt.Printf("Block %x failed verification\n", block.Hash)
		}
		
		if len(block.PrevHash) == 0 {
			break
		}
	}
	
	if isValid {
		fmt.Println("Blockchain validation successful. All blocks are valid.")
	} else {
		fmt.Println("Blockchain validation failed!")
	}
}

func (cli *CommandLine) setConsistency(level string) {
    chain := blockchain.ContinueBlockChain("")
    defer chain.Database.Close()
    
    switch level {
    case "strong":
        chain.ConsistencyMgr.SetConsistency(blockchain.StrongConsistency)
    case "causal":
        chain.ConsistencyMgr.SetConsistency(blockchain.CausalConsistency)
    case "eventual":
        chain.ConsistencyMgr.SetConsistency(blockchain.EventualConsistency)
    default:
        fmt.Println("Invalid consistency level")
        return
    }
    fmt.Println("Consistency level set to:", level)
}
func Handle(err error) {
    if err != nil {
        log.Panic(err)
    }
}
func (cli *CommandLine) setTrust(nodeID string, score float64) {
    chain := blockchain.ContinueBlockChain("")
    defer chain.Database.Close()
    
    // Update trust score in the blockchain
    chain.UpdateTrustScore(nodeID, score >= 0.5)
    fmt.Printf("Trust score for %s set to %.2f\n", nodeID, score)
}

func (cli *CommandLine) addFunds(address string, amount int) {
    if !wallet.ValidateAddress(address) {
        log.Panic("Address is not valid")
    }
    if amount <= 0 {
        log.Panic("Amount must be positive")
    }

    chain := blockchain.ContinueBlockChain("")
    defer chain.Database.Close()
    
    // Create transaction with explicit amount
    cbTx := blockchain.CoinbaseTx(address, fmt.Sprintf("Reward %d", amount), amount)
    chain.AddBlock([]*blockchain.Transaction{cbTx})
    fmt.Printf("Added %d coins to %s\n", amount, address)
}

///////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////

// Helper function to select address from list
func (cli *CommandLine) selectAddress(scanner *bufio.Scanner, prompt string) (string, bool) {
    addresses := cli.getAllAddresses()
    if len(addresses) == 0 {
        fmt.Println("No wallets found. Please create a wallet first.")
        return "", false
    }

    fmt.Println(prompt)
    for i, addr := range addresses {
        fmt.Printf("%d. %s\n", i+1, addr)
    }
    fmt.Printf("Choose address (1-%d or 0 to cancel): ", len(addresses))
    
    scanner.Scan()
    idx, err := strconv.Atoi(scanner.Text())
    if err != nil || idx < 0 || idx > len(addresses) {
        fmt.Println("Invalid selection")
        return "", false
    }
    if idx == 0 {
        return "", false // Cancelled
    }
    return addresses[idx-1], true
}

// Interactive version of getBalance
func (cli *CommandLine) getBalanceInteractive(scanner *bufio.Scanner) {
    address, ok := cli.selectAddress(scanner, "Select address to check balance:")
    if ok {
        cli.getBalance(address)
    }
}

// Interactive version of send
func (cli *CommandLine) sendInteractive(scanner *bufio.Scanner) {
    fmt.Println("\n=== Send Coins ===")
    from, ok1 := cli.selectAddress(scanner, "Select sender address:")
    if !ok1 {
        return
    }
    
    to, ok2 := cli.selectAddress(scanner, "Select recipient address:")
    if !ok2 {
        return
    }

    fmt.Print("Amount to send: ")
    scanner.Scan()
    amount, err := strconv.Atoi(scanner.Text())
    if err != nil {
        fmt.Println("Invalid amount")
        return
    }

    cli.send(from, to, amount)
}

// Interactive version of addFunds
func (cli *CommandLine) addFundsInteractive(scanner *bufio.Scanner) {
    address, ok := cli.selectAddress(scanner, "Select address to fund:")
    if !ok {
        return
    }

    fmt.Print("Amount to add: ")
    scanner.Scan()
    amount, err := strconv.Atoi(scanner.Text())
    if err != nil {
        fmt.Println("Invalid amount")
        return
    }

    cli.addFunds(address, amount)
}

func (cli *CommandLine) initializeBlockchainInteractive(scanner *bufio.Scanner) {
    fmt.Println("\n=== Initialize Blockchain ===")
    
    addresses := cli.getAllAddresses()
    if len(addresses) == 0 {
        fmt.Println("No wallets found. Please create a wallet first (option 1).")
        return
    }

    fmt.Println("Select address for genesis block:")
    for i, addr := range addresses {
        fmt.Printf("%d. %s\n", i+1, addr)
    }
    
    fmt.Printf("Choose address (1-%d or 0 to cancel): ", len(addresses))
    scanner.Scan()
    idx, err := strconv.Atoi(scanner.Text())
    
    if err != nil || idx < 0 || idx > len(addresses) {
        fmt.Println("Invalid selection")
        return
    }
    
    if idx == 0 {
        fmt.Println("Cancelled")
        return
    }

    address := addresses[idx-1]
    
    if blockchain.DBExists() {
        fmt.Println("Blockchain already exists. Use option 6 to view it.")
        return
    }

    fmt.Printf("\nCreating blockchain with address %s...\n", address)
    chain := blockchain.InitBlockChain(address)
    chain.Database.Close()
    fmt.Println("Blockchain created successfully with 100 initial coins!")
}

// getAllAddresses helper function
func (cli *CommandLine) getAllAddresses() []string {
    wallets, err := wallet.CreateWallets()
    if err != nil {
        return []string{}
    }
    return wallets.GetAllAddresses()
}

// waitForEnter helper function
func (cli *CommandLine) waitForEnter(scanner *bufio.Scanner, message string) {
    fmt.Print(message)
    scanner.Scan()
}
func (cli *CommandLine) setConsistencyInteractive(scanner *bufio.Scanner) {
    fmt.Println("\nSelect consistency level:")
    fmt.Println("1. Strong (slow, most secure)")
    fmt.Println("2. Causal (balanced)")
    fmt.Println("3. Eventual (fast, less secure)")
    fmt.Print("Your choice: ")
    
    scanner.Scan()
    var level string
    switch scanner.Text() {
    case "1":
        level = "strong"
    case "2":
        level = "causal"
    case "3":
        level = "eventual"
    default:
        fmt.Println("Invalid selection")
        return
    }
    
    chain := blockchain.ContinueBlockChain("")
    defer chain.Database.Close()
    
    switch level {
    case "strong":
        chain.ConsistencyMgr.SetConsistency(blockchain.StrongConsistency)
    case "causal":
        chain.ConsistencyMgr.SetConsistency(blockchain.CausalConsistency)
    case "eventual":
        chain.ConsistencyMgr.SetConsistency(blockchain.EventualConsistency)
    }
    fmt.Printf("Successfully set to %s consistency\n", level)
}

func (cli *CommandLine) setTrustInteractive(scanner *bufio.Scanner) {
    fmt.Print("\nEnter Node ID: ")
    scanner.Scan()
    nodeID := scanner.Text()
    
    fmt.Print("Enter Trust Score (0.0-1.0): ")
    scanner.Scan()
    score, err := strconv.ParseFloat(scanner.Text(), 64)
    
    if err != nil || score < 0 || score > 1 {
        fmt.Println("Invalid score (must be 0.0-1.0)")
        return
    }
    
    cli.setTrust(nodeID, score)
    fmt.Printf("Set trust score for %s to %.2f\n", nodeID, score)
}

func (cli *CommandLine) viewNetworkStatus() {
    chain := blockchain.ContinueBlockChain("")
    defer chain.Database.Close()
    
    fmt.Println("\n=== Network Status ===")
    fmt.Printf("Current Consistency: %v\n", chain.ConsistencyMgr.GetConsistencyLevel(chain.GetNodeID()))
    
    netStats := chain.GetNetworkStats()
    if netStats != nil {
        fmt.Printf("Partition Probability: %.2f%%\n", netStats.PredictPartitionProbability()*100)
    } else {
        fmt.Println("Network stats unavailable")
    }
    
    fmt.Println("\nTrust Scores:")
    for node, score := range chain.GetTrustScores() {
        fmt.Printf("- %s: %.2f\n", node, score)
    }
}

func (cli *CommandLine) printMenu() {
	if cli.NodeManagementMode {
        fmt.Println("\n=== Node Management ===")
        fmt.Println("1. Set Consistency Level")
        fmt.Println("2. Set Node Trust Score")
        fmt.Println("3. View Network Status")
        fmt.Println("4. Back to Main Menu")
    } else {
		fmt.Println("\n=== Blockchain CLI Menu ===")
		fmt.Println("1. Create Wallet")
		fmt.Println("2. List Addresses")
		fmt.Println("3. Initialize Blockchain")
		fmt.Println("4. Get Balance") 
		fmt.Println("5. Send Coins")
		fmt.Println("6. Print Chain")
		fmt.Println("7. Add Funds")
		fmt.Println("8. Validate Chain")
		fmt.Println("9. Node Management")
		fmt.Println("0. Exit")
	}
	fmt.Print("Select option: ")
}

func (cli *CommandLine) interactiveMode() {
    scanner := bufio.NewScanner(os.Stdin)
    
    for {
        cli.printMenu()
        scanner.Scan()
        choice := scanner.Text()
		 if cli.NodeManagementMode {
            switch choice {
            case "1":
                cli.setConsistencyInteractive(scanner)
            case "2":
                cli.setTrustInteractive(scanner)
            case "3":
                cli.viewNetworkStatus()
            case "4":
                cli.NodeManagementMode = false
            default:
                fmt.Println("Invalid option")
            }
        } else {
        
			switch choice {
			case "1":
				cli.createWallet()
			case "2":
				cli.listAddresses()
			case "3":
				cli.initializeBlockchainInteractive(scanner)
			case "4":
				cli.getBalanceInteractive(scanner)
			case "5":
				cli.sendInteractive(scanner)
			case "6":
				cli.printChain()
			case "7":
				cli.addFundsInteractive(scanner)
			case "8":
				cli.validateChain()
			case "9":
                cli.NodeManagementMode = true
            case "0":
                os.Exit(0)

			default:
				fmt.Println("Invalid option")
			}
		}
        cli.waitForEnter(scanner, "Press Enter to continue...")
    }
}

func (cli *CommandLine) Run() {
	
    // Handle global flags
    if len(os.Args) > 1 {
        for i, arg := range os.Args {
            switch arg {
            case "-consistency":
                cli.Consistency = os.Args[i+1]
                os.Args = append(os.Args[:i], os.Args[i+2:]...)
            case "-node":
                cli.NodeID = os.Args[i+1]
                os.Args = append(os.Args[:i], os.Args[i+2:]...)
            case "-trustscore":
                score, err := strconv.ParseFloat(os.Args[i+1], 64)
                Handle(err)
                cli.TrustScore = score
                os.Args = append(os.Args[:i], os.Args[i+2:]...)
            }
        }
    }
	
	// If no commands left after processing flags, go interactive
    if len(os.Args) < 2 {
        cli.interactiveMode()
        return
    }

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	validateChainCmd := flag.NewFlagSet("validatechain", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "validatechain":
		err := validateChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "setconsistency":
		consistencyCmd := flag.NewFlagSet("setconsistency", flag.ExitOnError)
		level := consistencyCmd.String("level", "", "Consistency level (strong/eventual)")
		err := consistencyCmd.Parse(os.Args[2:])
		Handle(err)
		
		if *level == "" {
			log.Panic("Please specify consistency level (strong/eventual)")
		}
		
		chain := blockchain.ContinueBlockChain("")
		
		switch *level {
		case "strong":
			chain.ConsistencyMgr.SetConsistency(blockchain.StrongConsistency)
		case "eventual":
			chain.ConsistencyMgr.SetConsistency(blockchain.EventualConsistency)
		default:
			chain.Database.Close()
			log.Panic("Invalid consistency level")
		}
		
		chain.Database.Close()
		fmt.Println("Consistency level set to:", *level)
		return
	case "settrust":
		trustCmd := flag.NewFlagSet("settrust", flag.ExitOnError)
		node := trustCmd.String("node", "", "Node ID")
		score := trustCmd.Float64("score", 0.5, "Trust score")
		err := trustCmd.Parse(os.Args[2:])
		Handle(err)
		cli.setTrust(*node, *score)
		runtime.Goexit()
	case "addfunds":
		addFundsCmd := flag.NewFlagSet("addfunds", flag.ExitOnError)
		addFundsAddress := addFundsCmd.String("address", "", "Address to fund")
		addFundsAmount := addFundsCmd.Int("amount", 0, "Amount to add")
		err := addFundsCmd.Parse(os.Args[2:])
		Handle(err)
		cli.addFunds(*addFundsAddress, *addFundsAmount)
	
	default:
		//cli.printUsage()
		cli.interactiveMode()
		runtime.Goexit()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}

		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
	
	if createWalletCmd.Parsed() {
		cli.createWallet()
	}
	
	if listAddressesCmd.Parsed() {
		cli.listAddresses()
	}
	
	if validateChainCmd.Parsed() {
		cli.validateChain()
	}
}