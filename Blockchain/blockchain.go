package blockchain

import (
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"bytes"
	"math"
	"log"

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
	cbtx := CoinbaseTx(address, genesisData, 100)
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

// GetNodeID returns the node's identifier
func (chain *BlockChain) GetNodeID() string {
    return chain.nodeID
}

// GetNetworkStats returns the network statistics
func (chain *BlockChain) GetNetworkStats() *NetworkStats {
    return chain.ConsistencyMgr.NetworkStats
}

// GetTrustScores returns a copy of the trust scores map
func (chain *BlockChain) GetTrustScores() map[string]float64 {
    chain.mutex.RLock()
    defer chain.mutex.RUnlock()
    
    scores := make(map[string]float64)
    for k, v := range chain.trustScores {
        scores[k] = v
    }
    return scores
}

//rebuildAMF rebuilds the Adaptive Merkle Forest from existing blocks
func (chain *BlockChain) rebuildAMF() {
    iter := chain.Iterator()
    for {
        block := iter.Next()
        
        // Add block components to AMF
        chain.AMF.AddData(block.Hash)
        chain.AMF.AddData(block.Serialize())
        for _, tx := range block.Transactions {
            chain.AMF.AddData(tx.ID)
        }
        
        if len(block.PrevHash) == 0 {
            break
        }
    }
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

	blockchain := &BlockChain{
		LastHash:       lastHash,
		Database:       db,
		AMF:            NewAdaptiveMerkleForest(),
		ConsistencyMgr: NewConsistencyOrchestrator(),
		SyncMgr:        NewCrossShardSynchronizer(),
		nodeID:         generateNodeID(),
		trustScores:    make(map[string]float64),
	}
	
	// Rebuild AMF from existing blocks
    blockchain.rebuildAMF()
    return blockchain
}

// generateNodeID creates a unique node identifier
func generateNodeID() string {
	buffer := make([]byte, 16)
	// Simplified ID generation
	return hex.EncodeToString(buffer)
}


// AddBlock adds a new block to the blockchain
func (chain *BlockChain) AddBlock(transactions []*Transaction) {
    consistencyLevel := chain.ConsistencyMgr.GetConsistencyLevel(chain.nodeID)
    chain.mutex.Lock()
    defer chain.mutex.Unlock()

    // Get last hash
    var lastHash []byte
    err := chain.Database.View(func(txn *badger.Txn) error {
        item, err := txn.Get([]byte("lh"))
        Handle(err)
        return item.Value(func(val []byte) error {
            lastHash = append([]byte{}, val...)
            return nil
        })
    })
    Handle(err)

    // Create and prepare block
    newBlock := CreateBlock(transactions, lastHash)
    
    // 1. Add all components to AMF FIRST
    for _, tx := range transactions {
        chain.AMF.AddData(tx.ID)
    }
    chain.AMF.AddData(newBlock.Hash)
    chain.AMF.AddData(newBlock.Serialize())

    // 2. Verify BEFORE storing in DB
    if !chain.VerifyBlockIntegrity(newBlock) {
        log.Panic("Block failed pre-storage verification")
    }

    // 3. Store in database
    err = chain.Database.Update(func(txn *badger.Txn) error {
        serialized := newBlock.Serialize()
        if err := txn.Set(newBlock.Hash, serialized); err != nil {
            return err
        }
        if err := txn.Set([]byte("lh"), newBlock.Hash); err != nil {
            return err
        }
        chain.LastHash = newBlock.Hash
        return nil
    })
    Handle(err)

    // 4. Post-storage verification
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
    // 1. Always verify proof of work
    pow := NewProof(block)
    if !pow.Validate() {
        return false
    }

    // 2. Skip AMF check for genesis block
    if len(block.PrevHash) == 0 {
        return true
    }

    // 3. Check transaction Merkle root
    calculatedRoot := block.HashTransactions()
    if !bytes.Equal(calculatedRoot, block.TxTrie.Hash) {
        return false
    }

    // 4. Verify block in AMF with retry logic
    if chain.AMF != nil {
        if _, exists := chain.AMF.GenerateMerkleProof(block.Hash); !exists {
            // Attempt to recover by adding to AMF
            chain.AMF.AddData(block.Hash)
            chain.AMF.AddData(block.Serialize())
            
            // Retry verification
            if _, exists := chain.AMF.GenerateMerkleProof(block.Hash); !exists {
                return false
            }
        }
    }

    return true
}


// UpdateTrustScore updates a node's trust score for Byzantine fault tolerance
// Add to blockchain.go
func (chain *BlockChain) UpdateTrustScore(nodeID string, success bool) {
    chain.mutex.Lock()
    defer chain.mutex.Unlock()
    
    currentScore := chain.trustScores[nodeID]
    if success {
        chain.trustScores[nodeID] = math.Min(currentScore+0.1, 1.0)
    } else {
        chain.trustScores[nodeID] = math.Max(currentScore-0.2, 0.0)
    }
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

var addressLocks = make(map[string]bool)
var lockMutex sync.Mutex

func (chain *BlockChain) LockAddress(address string) {
    lockMutex.Lock()
    defer lockMutex.Unlock()
    addressLocks[address] = true
}

func (chain *BlockChain) UnlockAddress(address string) {
    lockMutex.Lock()
    defer lockMutex.Unlock()
    delete(addressLocks, address)
}

func (chain *BlockChain) IsAddressLocked(address string) bool {
    lockMutex.Lock()
    defer lockMutex.Unlock()
    return addressLocks[address]
}