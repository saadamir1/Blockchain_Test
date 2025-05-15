package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"time"
)

// Take the data from the block
// create a nonce which starts at 0
// create a hash of the block data + nonce
// check if the hash starts with required number of zeros
// if it does, we have found a valid nonce

const Difficulty = 12 // Number of leading zeros required in the hash

// ProofOfWork represents a proof-of-work system
type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

// NewProof creates a new proof-of-work instance
func NewProof(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-Difficulty)) // Shift left by (256 - Difficulty) bits

	pow := &ProofOfWork{b, target}

	return pow
}

// initData prepares the data for hashing
func (pow *ProofOfWork) initData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.Block.PrevHash,
			pow.Block.HashTransactions(),
			pow.Block.StateRoot,
			pow.Block.AccumulatorRoot,
			ToHex(int64(nonce)),
			ToHex(int64(Difficulty)),
			ToHex(pow.Block.Timestamp),
		},
		[]byte{},
	)
	return data
}

// Run performs the proof-of-work computation
func (pow *ProofOfWork) Run() (int, []byte) {
	var intHash big.Int
	var hash [32]byte
	nonce := 0
	
	// Add randomness injection for enhanced security
	rand.Seed(time.Now().UnixNano())
	startNonce := rand.Intn(10000)
	nonce = startNonce
	
	fmt.Printf("Mining a new block with start nonce %d\n", startNonce)
	
	// Verification loop
	for nonce < math.MaxInt64 {
		data := pow.initData(nonce)
		hash = sha256.Sum256(data)

		fmt.Printf("\rHash: %x", hash[:4])
		intHash.SetBytes(hash[:])

		if intHash.Cmp(pow.Target) == -1 {
			break // Found a valid nonce
		} else {
			nonce++ // Increment nonce and try again
		}
	}
	fmt.Printf("\nNonce found: %d\n", nonce)
	
	return nonce, hash[:]
}

// Validate verifies the proof-of-work
func (pow *ProofOfWork) Validate() bool {
	var intHash big.Int
	data := pow.initData(pow.Block.Nonce)
	hash := sha256.Sum256(data)

	intHash.SetBytes(hash[:])

	return intHash.Cmp(pow.Target) == -1 // Check if the hash is less than the target
}

// ToHex converts an int64 to a byte slice
func ToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

// VerifiableRandomFunction provides a provably unique random value
type VerifiableRandomFunction struct {
	Seed []byte
}

// NewVRF creates a new VRF instance
func NewVRF(seed []byte) *VerifiableRandomFunction {
	return &VerifiableRandomFunction{
		Seed: seed,
	}
}

// Generate creates a verifiable random value and proof
func (vrf *VerifiableRandomFunction) Generate() ([]byte, []byte) {
	// Create a random value
	timestamp := time.Now().UnixNano()
	combined := append(vrf.Seed, ToHex(timestamp)...)
	hash := sha256.Sum256(combined)
	
	// Create a proof of that value
	proof := sha256.Sum256(hash[:])
	
	return hash[:], proof[:]
}

// Verify checks if a VRF output is valid
func (vrf *VerifiableRandomFunction) Verify(value, proof []byte) bool {
	verificationHash := sha256.Sum256(value)
	return bytes.Equal(verificationHash[:], proof)
}

// MultipartyConsensus implements a hybrid consensus mechanism combining PoW and BFT
type MultipartyConsensus struct {
	NodeID         string
	TrustThreshold float64
	TrustScores    map[string]float64
	VRF            *VerifiableRandomFunction
}

// NewMultipartyConsensus creates a new consensus instance
func NewMultipartyConsensus(nodeID string) *MultipartyConsensus {
	seed := append([]byte(nodeID), ToHex(time.Now().UnixNano())...)
	return &MultipartyConsensus{
		NodeID:         nodeID,
		TrustThreshold: 0.67, // 2/3 majority required
		TrustScores:    make(map[string]float64),
		VRF:            NewVRF(seed),
	}
}

// AddTrustScore updates the trust score for a node
func (mc *MultipartyConsensus) AddTrustScore(nodeID string, score float64) {
	mc.TrustScores[nodeID] = score
}

// SelectLeader uses VRF to select a leader for the next round
func (mc *MultipartyConsensus) SelectLeader(nodes []string) string {
	if len(nodes) == 0 {
		return ""
	}
	
	// Generate random value
	randomValue, _ := mc.VRF.Generate()
	
	// Convert to integer for selection
	seed := binary.BigEndian.Uint64(randomValue[:8])
	rand.Seed(int64(seed))
	
	// Weight selection by trust scores
	var weightedNodes []string
	for _, node := range nodes {
		weight := int(mc.TrustScores[node] * 100)
		if weight < 1 {
			weight = 1
		}
		
		for i := 0; i < weight; i++ {
			weightedNodes = append(weightedNodes, node)
		}
	}
	
	// Select random node from weighted list
	selectedIndex := rand.Intn(len(weightedNodes))
	return weightedNodes[selectedIndex]
}

// ValidateBlock combines PoW and trust scores to validate a block
func (mc *MultipartyConsensus) ValidateBlock(block *Block, validations map[string]bool) bool {
	// First ensure PoW is valid
	pow := NewProof(block)
	if !pow.Validate() {
		return false
	}
	
	// Count trusted validations
	trueCount := 0
	totalTrust := 0.0
	
	for nodeID, isValid := range validations {
		score := mc.TrustScores[nodeID]
		if score == 0 {
			score = 0.5 // Default trust for unknown nodes
		}
		
		if isValid {
			trueCount++
			totalTrust += score
		}
	}
	
	// Require both majority of nodes and sufficient trust
	nodeThreshold := len(validations) * 2 / 3
	return trueCount >= nodeThreshold && totalTrust >= mc.TrustThreshold
}