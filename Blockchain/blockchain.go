package blockchain

import (
	"encoding/hex"
	"fmt"
	"os"
	"sync"

	"github.com/dgraph-io/badger"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

// BlockChain represents the blockchain data structure with enhanced features
type BlockChain struct {
	LastHash       []byte
	Database       *badger.DB
	AMF            *AdaptiveMerkleForest    // For Adaptive Merkle Forest requirement
	ConsistencyMgr *ConsistencyOrchestrator // For CAP Theorem Dynamic Optimization
	SyncMgr        *CrossShardSynchronizer  // For Cross-Shard State Synchronization
	mutex          sync.RWMutex
	nodeID         string
	trustScores    map[string]float64 // For Byzantine Fault Tolerance
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

// InitBlockChain creates a new blockchain with genesis block
func InitBlockChain(address string) *BlockChain {
	var lastHash []byte

	if DBExists() {
		fmt.Println("Blockchain already exists")
		os.Exit(1)
	}

	// Database initialization logic
	opts := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opts)
	Handle(err)
	
	// Create genesis block
	cbtx := CoinbaseTx(address, genesisData)
	genesis := Genesis(cbtx)
	
	// Store genesis block in database
	err = db.Update(func(txn *badger.Txn) error {
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash
		return err
	})
	Handle(err)

	// Create blockchain with advanced components
	blockchain := BlockChain{
		LastHash:       lastHash,
		Database:       db,
		AMF:            NewAdaptiveMerkleForest(),
		ConsistencyMgr: NewConsistencyOrchestrator(),
		SyncMgr:        NewCrossShardSynchronizer(),
		nodeID:         generateNodeID(),
		trustScores:    make(map[string]float64),
	}
	
	// Add genesis block data to AMF
	blockchain.AMF.AddData(lastHash)
	
	return &blockchain
}

// ContinueBlockChain continues an existing blockchain
func ContinueBlockChain(address string) *BlockChain {
	if !DBExists() {
		fmt.Println("No blockchain found, create one first")
		os.Exit(1)
	}

	var lastHash []byte

	opts := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opts)
	Handle(err)

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
		return err
	})
	Handle(err)

	blockchain := BlockChain{
		LastHash:       lastHash,
		Database:       db,
		AMF:            NewAdaptiveMerkleForest(),
		ConsistencyMgr: NewConsistencyOrchestrator(),
		SyncMgr:        NewCrossShardSynchronizer(),
		nodeID:         generateNodeID(),
		trustScores:    make(map[string]float64),
	}

	return &blockchain
}

// generateNodeID creates a unique node identifier
func generateNodeID() string {
	buffer := make([]byte, 16)
	// Simplified ID generation
	return hex.EncodeToString(buffer)
}

// AddBlock adds a new block to the blockchain
func (chain *BlockChain) AddBlock(transactions []*Transaction) {
	// Get dynamic consistency level
	consistencyLevel := chain.ConsistencyMgr.GetConsistencyLevel(chain.nodeID)
	
	chain.mutex.Lock()
	defer chain.mutex.Unlock()
	
	// Get the last hash
	var lastHash []byte
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
		})
		return err
	})
	Handle(err)

	// Create new block
	newBlock := CreateBlock(transactions, lastHash)
	
	// Update AMF (Adaptive Merkle Forest)
	for _, tx := range transactions {
		chain.AMF.AddData(tx.ID)
	}
	
	// Store block in database
	err = chain.Database.Update(func(txn *badger.Txn) error {
		serializedBlock := newBlock.Serialize()
		err := txn.Set(newBlock.Hash, serializedBlock)
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		chain.AMF.AddData(serializedBlock)
		return err
	})
	Handle(err)
	
	// Handle cross-shard synchronization based on consistency level
	if consistencyLevel == StrongConsistency {
		chain.SyncMgr.CreateCommitment(hex.EncodeToString(newBlock.Hash), newBlock.Serialize())
	}
}

// Iterator returns a blockchain iterator
func (chain *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{chain.LastHash, chain.Database}
}

// Next returns the next block in the blockchain
func (iter *BlockChainIterator) Next() *Block {
	var block *Block
	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		err = item.Value(func(val []byte) error {
			block = Deserialize(val)
			return nil
		})
		return err
	})
	Handle(err)
	iter.CurrentHash = block.PrevHash
	return block
}

// FindUnspentTransactions finds all unspent transactions for an address
func (chain *BlockChain) FindUnspentTransactions(address string) []*Transaction {
	var unspentTxs []*Transaction

	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
						inTxID := hex.EncodeToString(in.ID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.OutIndex)
					}
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTxs
}

// FindUTXO finds all unspent transaction outputs for an address
func (chain *BlockChain) FindUTXO(address string) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := chain.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

// FindSpendableOutputs finds spendable outputs for a transaction
func (chain *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(address)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts
}

// VerifyBlockIntegrity performs advanced verification of a block
func (chain *BlockChain) VerifyBlockIntegrity(block *Block) bool {
	// Verify proof of work
	pow := NewProof(block)
	if !pow.Validate() {
		return false
	}
	
	// Verify block exists in AMF
	_, exists := chain.AMF.GenerateMerkleProof(block.Serialize())
	return exists
}

// UpdateTrustScore updates a node's trust score for Byzantine fault tolerance
func (chain *BlockChain) UpdateTrustScore(nodeID string, success bool) {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()
	
	currentScore, exists := chain.trustScores[nodeID]
	if !exists {
		currentScore = 0.5 // Default trust score
	}
	
	// Adjust score based on node behavior
	if success {
		currentScore = min(currentScore + 0.1, 1.0)
	} else {
		currentScore = max(currentScore - 0.2, 0.0)
	}
	
	chain.trustScores[nodeID] = currentScore
}

// DBExists checks if the blockchain database exists
func DBExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// Helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}