package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// Transaction represents a blockchain transaction with enhanced features
type Transaction struct {
	ID          []byte
	Inputs      []TxInput
	Outputs     []TxOutput
	Timestamp   int64
	Signature   []byte
	LockTime    int64
	StateProof  []byte // Zero-knowledge proof for state validation
}

// SetID calculates the transaction ID
func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

// CoinbaseTx creates a new coinbase transaction
func CoinbaseTx(to string, data string, amount int) *Transaction {
    if data == "" {
        data = fmt.Sprintf("Reward %d to %s", amount, to)
    }

    txIn := TxInput{
        ID:        []byte{},
        OutIndex:  -1,
        Signature: data,
        StateProof: nil,
    }

    txOut := TxOutput{
        Value:    amount, 
        PubKey:   to,
        Metadata: nil,
    }

    tx := Transaction{
        ID:         nil,
        Inputs:     []TxInput{txIn},
        Outputs:    []TxOutput{txOut},
        Timestamp:  time.Now().Unix(),
        Signature:  nil,
        LockTime:   0,
        StateProof: nil,
    }
    tx.SetID()

    return &tx
}

// IsCoinbase checks if this is a coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].OutIndex == -1
}

// NewTransaction creates a new transaction
func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
    // Check for pending spends
    if chain.IsAddressLocked(from) {
        log.Panic("ERROR: Address has pending transaction (double-spend attempt)")
    }

    // Lock the address until transaction is mined
    chain.LockAddress(from)
    defer chain.UnlockAddress(from)

    var Inputs []TxInput
    var Outputs []TxOutput

    acc, validOutputs := chain.FindSpendableOutputs(from, amount)
    if acc < amount {
        chain.UnlockAddress(from)
        log.Panic("ERROR: Not enough funds")
    }

    // Build inputs
    for txid, outs := range validOutputs {
        txID, err := hex.DecodeString(txid)
        Handle(err)

        for _, out := range outs {
            input := TxInput{
                ID:        txID,
                OutIndex:  out,
                Signature: from,
                StateProof: nil,
            }
            Inputs = append(Inputs, input)
        }
    }

    // Build outputs
    Outputs = append(Outputs, TxOutput{
        Value:    amount,
        PubKey:   to,
        Metadata: nil,
    })

    // Change output
    if acc > amount {
        Outputs = append(Outputs, TxOutput{
            Value:    acc - amount,
            PubKey:   from,
            Metadata: nil,
        })
    }

    tx := Transaction{
        ID:         nil,
        Inputs:     Inputs,
        Outputs:    Outputs,
        Timestamp:  time.Now().Unix(),
        Signature:  nil,
        LockTime:   0,
        StateProof: nil,
    }
    tx.SetID()
    
    // Generate state proof
    proof := sha256.Sum256(tx.ID)
    tx.StateProof = proof[:]

    return &tx
}

// VerifyTransaction verifies transaction integrity
func VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	
	// Verify signatures (simplified in this implementation)
	for _, input := range tx.Inputs {
		if input.Signature == "" {
			return false
		}
	}
	
	// Verify state proof
	if tx.StateProof == nil {
		return false
	}
	
	expected := sha256.Sum256(tx.ID)
	return bytes.Equal(tx.StateProof, expected[:])
}

// CreateStateProof generates a zero-knowledge proof for a transaction
func (tx *Transaction) CreateStateProof() []byte {
	if tx.StateProof != nil {
		return tx.StateProof
	}
	
	// Simple hash-based proof (in a real ZK implementation, this would be more complex)
	proof := sha256.Sum256(append(tx.ID, byte(tx.Timestamp)))
	tx.StateProof = proof[:]
	
	return tx.StateProof
}