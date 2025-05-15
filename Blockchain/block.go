package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"time"
)

// Block represents a block in the blockchain with enhanced features
type Block struct {
	Hash           []byte
	Transactions   []*Transaction
	PrevHash       []byte
	Nonce          int
	Timestamp      int64
	StateRoot      []byte // Root hash of state trie
	AccumulatorRoot []byte // Root of cryptographic accumulator
	TxTrie         *MerkleNode // Multi-level Merkle structure for transactions
	EntropyValue   float64 // Entropy-based validation metric
}

// HashTransactions creates a Merkle tree of transaction hashes
func (b *Block) HashTransactions() []byte {
	if b.TxTrie != nil {
		return b.TxTrie.Hash
	}
	
	var txHashes [][]byte
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	
	// Convert transactions to Merkle tree nodes
	var nodes []*MerkleNode
	for _, hash := range txHashes {
		nodes = append(nodes, &MerkleNode{Hash: hash})
	}
	
	// Build Merkle tree
	for len(nodes) > 1 {
		var level []*MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				combinedHash := append(nodes[i].Hash, nodes[i+1].Hash...)
				hash := sha256.Sum256(combinedHash)
				level = append(level, &MerkleNode{
					Hash:  hash[:],
					Left:  nodes[i],
					Right: nodes[i+1],
				})
			} else {
				level = append(level, nodes[i])
			}
		}
		nodes = level
	}
	
	if len(nodes) > 0 {
		b.TxTrie = nodes[0]
		return b.TxTrie.Hash
	}
	
	// Default hash if no transactions
	hash := sha256.Sum256([]byte{})
	return hash[:]
}

func CreateBlock(txs []*Transaction, prevHash []byte) *Block {
	block := &Block{
		Hash:          []byte{},
		Transactions:  txs,
		PrevHash:      prevHash,
		Nonce:         0,
		Timestamp:     time.Now().Unix(),
		StateRoot:     []byte{},
		EntropyValue:  calculateBlockEntropy(txs),
	}
	
	// Create transaction trie and get root hash
	txRoot := block.HashTransactions()
	
	// Create state accumulator root (simplified)
	accumulatorRoot := sha256.Sum256(append(txRoot, prevHash...))
	block.AccumulatorRoot = accumulatorRoot[:]
	
	// Create state root (simplified)
	stateRoot := sha256.Sum256(append(accumulatorRoot[:], []byte(string(block.Timestamp))...))
	block.StateRoot = stateRoot[:]
	
	// Run proof of work
	pow := NewProof(block)
	nonce, hash := pow.Run()
	
	block.Hash = hash[:]
	block.Nonce = nonce
	
	return block
}

// calculateBlockEntropy computes entropy-based validation metric
func calculateBlockEntropy(txs []*Transaction) float64 {
	if len(txs) == 0 {
		return 0.0
	}
	
	// Simple entropy calculation (can be enhanced)
	uniqueInputs := make(map[string]bool)
	uniqueOutputs := make(map[string]bool)
	
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			uniqueInputs[string(in.ID)] = true
		}
		for _, out := range tx.Outputs {
			uniqueOutputs[out.PubKey] = true
		}
	}
	
	// Entropy based on unique inputs and outputs
	inputEntropy := float64(len(uniqueInputs)) / float64(len(txs))
	outputEntropy := float64(len(uniqueOutputs)) / float64(len(txs))
	
	return (inputEntropy + outputEntropy) / 2.0
}

func Genesis(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{})
}

func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)
	
	err := encoder.Encode(b)
	Handle(err)
	
	return res.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	
	err := decoder.Decode(&block)
	Handle(err)
	
	return &block
}

// PruneBlockData compresses block data for archival purposes
func PruneBlockData(b *Block) *Block {
	// Create a pruned copy of the block with minimal data
	pruned := &Block{
		Hash:          b.Hash,
		PrevHash:      b.PrevHash,
		Nonce:         b.Nonce,
		Timestamp:     b.Timestamp,
		StateRoot:     b.StateRoot,
		AccumulatorRoot: b.AccumulatorRoot,
		EntropyValue:  b.EntropyValue,
	}
	
	// Only keep transaction IDs for Merkle verification
	var txIDs []*Transaction
	for _, tx := range b.Transactions {
		txIDs = append(txIDs, &Transaction{ID: tx.ID})
	}
	pruned.Transactions = txIDs
	
	return pruned
}

func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}