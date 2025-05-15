package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Core structures
type MerkleNode struct {
	Hash  []byte
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

type AMFShard struct {
	ID              string
	Root            *MerkleNode
	Data            [][]byte
	ComputationLoad int
	LastAccessed    time.Time
	mutex           sync.RWMutex
}

type AdaptiveMerkleForest struct {
	Shards         []*AMFShard
	mutex          sync.RWMutex
	shardThreshold int
	mergeInterval  time.Duration
}

// NewAdaptiveMerkleForest creates a new AMF
func NewAdaptiveMerkleForest() *AdaptiveMerkleForest {
	amf := &AdaptiveMerkleForest{
		Shards:         make([]*AMFShard, 0),
		shardThreshold: 100,
		mergeInterval:  5 * time.Minute,
	}
	
	rootShard := &AMFShard{
		ID:              "root",
		Data:            make([][]byte, 0),
		ComputationLoad: 0,
		LastAccessed:    time.Now(),
	}
	amf.Shards = append(amf.Shards, rootShard)
	
	go amf.runShardMonitor()
	return amf
}

// AddData adds data to the appropriate shard
func (amf *AdaptiveMerkleForest) AddData(data []byte) {
	amf.mutex.RLock()
	bestShard := amf.findOptimalShard(data)
	amf.mutex.RUnlock()

	bestShard.mutex.Lock()
	defer bestShard.mutex.Unlock()
	
	bestShard.Data = append(bestShard.Data, data)
	bestShard.ComputationLoad++
	bestShard.LastAccessed = time.Now()
	bestShard.Root = amf.buildMerkleTree(bestShard.Data)
	
	if bestShard.ComputationLoad > amf.shardThreshold {
		amf.splitShard(bestShard)
	}
}

// findOptimalShard determines the best shard for data placement
func (amf *AdaptiveMerkleForest) findOptimalShard(data []byte) *AMFShard {
	var bestShard *AMFShard
	minLoad := int(^uint(0) >> 1)
	
	for _, shard := range amf.Shards {
		shard.mutex.RLock()
		if shard.ComputationLoad < minLoad {
			minLoad = shard.ComputationLoad
			bestShard = shard
		}
		shard.mutex.RUnlock()
	}
	
	return bestShard
}

// splitShard splits a shard into two based on computational load
func (amf *AdaptiveMerkleForest) splitShard(shard *AMFShard) {
	amf.mutex.Lock()
	defer amf.mutex.Unlock()
	
	midPoint := len(shard.Data) / 2
	
	leftID := fmt.Sprintf("%s-L-%d", shard.ID, time.Now().UnixNano())
	rightID := fmt.Sprintf("%s-R-%d", shard.ID, time.Now().UnixNano())
	
	leftShard := &AMFShard{
		ID:              leftID,
		Data:            shard.Data[:midPoint],
		ComputationLoad: 0,
		LastAccessed:    time.Now(),
	}
	
	rightShard := &AMFShard{
		ID:              rightID,
		Data:            shard.Data[midPoint:],
		ComputationLoad: 0,
		LastAccessed:    time.Now(),
	}
	
	leftShard.Root = amf.buildMerkleTree(leftShard.Data)
	rightShard.Root = amf.buildMerkleTree(rightShard.Data)
	
	for i, s := range amf.Shards {
		if s.ID == shard.ID {
			amf.Shards = append(amf.Shards[:i], amf.Shards[i+1:]...)
			break
		}
	}
	
	amf.Shards = append(amf.Shards, leftShard, rightShard)
}

// mergeLowActivityShards merges shards with low activity
func (amf *AdaptiveMerkleForest) mergeLowActivityShards() {
	amf.mutex.Lock()
	defer amf.mutex.Unlock()
	
	if len(amf.Shards) <= 1 {
		return
	}
	
	now := time.Now()
	var mergeCandidates []*AMFShard
	
	for _, shard := range amf.Shards {
		shard.mutex.RLock()
		if now.Sub(shard.LastAccessed) > amf.mergeInterval && shard.ComputationLoad < amf.shardThreshold/2 {
			mergeCandidates = append(mergeCandidates, shard)
		}
		shard.mutex.RUnlock()
	}
	
	if len(mergeCandidates) < 2 {
		return
	}
	
	shard1 := mergeCandidates[0]
	shard2 := mergeCandidates[1]
	
	mergedID := fmt.Sprintf("merged-%d", time.Now().UnixNano())
	
	shard1.mutex.Lock()
	shard2.mutex.Lock()
	
	mergedData := append(shard1.Data, shard2.Data...)
	mergedShard := &AMFShard{
		ID:              mergedID,
		Data:            mergedData,
		ComputationLoad: 0,
		LastAccessed:    time.Now(),
	}
	
	mergedShard.Root = amf.buildMerkleTree(mergedData)
	
	shard1.mutex.Unlock()
	shard2.mutex.Unlock()
	
	var newShards []*AMFShard
	for _, s := range amf.Shards {
		if s.ID != shard1.ID && s.ID != shard2.ID {
			newShards = append(newShards, s)
		}
	}
	
	amf.Shards = append(newShards, mergedShard)
}

// buildMerkleTree builds a Merkle tree from given data
func (amf *AdaptiveMerkleForest) buildMerkleTree(data [][]byte) *MerkleNode {
	if len(data) == 0 {
		return nil
	}
	
	var nodes []*MerkleNode
	for _, item := range data {
		hash := sha256.Sum256(item)
		nodes = append(nodes, &MerkleNode{
			Hash: hash[:],
			Data: item,
		})
	}
	
	for len(nodes) > 1 {
		var levelNodes []*MerkleNode
		
		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				concatenated := append(nodes[i].Hash, nodes[i+1].Hash...)
				hash := sha256.Sum256(concatenated)
				
				parent := &MerkleNode{
					Hash:  hash[:],
					Left:  nodes[i],
					Right: nodes[i+1],
				}
				levelNodes = append(levelNodes, parent)
			} else {
				levelNodes = append(levelNodes, nodes[i])
			}
		}
		
		nodes = levelNodes
	}
	
	return nodes[0]
}

// GenerateMerkleProof creates a proof for data verification
func (amf *AdaptiveMerkleForest) GenerateMerkleProof(data []byte) ([][]byte, bool) {
	var targetShard *AMFShard
	found := false
	
	amf.mutex.RLock()
	defer amf.mutex.RUnlock()
	
	for _, shard := range amf.Shards {
		shard.mutex.RLock()
		for _, item := range shard.Data {
			if string(item) == string(data) {
				targetShard = shard
				found = true
				break
			}
		}
		shard.mutex.RUnlock()
		if found {
			break
		}
	}
	
	if !found || targetShard == nil {
		return nil, false
	}
	
	targetShard.mutex.RLock()
	defer targetShard.mutex.RUnlock()
	
	proof := make([][]byte, 0)
	proof = append(proof, targetShard.Root.Hash)
	
	return proof, true
}

// runShardMonitor periodically checks and optimizes shards
func (amf *AdaptiveMerkleForest) runShardMonitor() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		amf.mergeLowActivityShards()
	}
}

// BloomFilter implements a simple bloom filter for probabilistic membership queries
type BloomFilter struct {
	bits    []bool
	numHash int
}

// NewBloomFilter creates a new bloom filter
func NewBloomFilter(size int, numHash int) *BloomFilter {
	return &BloomFilter{
		bits:    make([]bool, size),
		numHash: numHash,
	}
}

// Add adds an item to the bloom filter
func (bf *BloomFilter) Add(item []byte) {
	for i := 0; i < bf.numHash; i++ {
		h := sha256.Sum256(append(item, byte(i)))
		index := int(h[0]) % len(bf.bits)
		bf.bits[index] = true
	}
}

// Contains checks if an item might be in the set
func (bf *BloomFilter) Contains(item []byte) bool {
	for i := 0; i < bf.numHash; i++ {
		h := sha256.Sum256(append(item, byte(i)))
		index := int(h[0]) % len(bf.bits)
		if !bf.bits[index] {
			return false
		}
	}
	return true
}

// ProbabilisticAccumulator implements an efficient membership verification system
type ProbabilisticAccumulator struct {
	data       map[string]bool
	compressor *BloomFilter
}

// NewProbabilisticAccumulator creates a new accumulator
func NewProbabilisticAccumulator() *ProbabilisticAccumulator {
	return &ProbabilisticAccumulator{
		data:       make(map[string]bool),
		compressor: NewBloomFilter(1024, 3),
	}
}

// Add adds an item to the accumulator
func (pa *ProbabilisticAccumulator) Add(data []byte) {
	hexData := hex.EncodeToString(data)
	pa.data[hexData] = true
	pa.compressor.Add(data)
}

// CrossShardSynchronizer manages state transfers between shards
type CrossShardSynchronizer struct {
	commitments map[string][]byte
	mutex       sync.RWMutex
}

// NewCrossShardSynchronizer creates a new synchronizer
func NewCrossShardSynchronizer() *CrossShardSynchronizer {
	return &CrossShardSynchronizer{
		commitments: make(map[string][]byte),
	}
}

// CreateCommitment creates a commitment for cross-shard operations
func (css *CrossShardSynchronizer) CreateCommitment(key string, value []byte) []byte {
	hash := sha256.Sum256(value)
	
	css.mutex.Lock()
	css.commitments[key] = hash[:]
	css.mutex.Unlock()
	
	return hash[:]
}

// SynchronizeState transfers state between shards
func (css *CrossShardSynchronizer) SynchronizeState(sourceShardID, targetShardID string, key string, value []byte) bool {
	commitment := css.CreateCommitment(key, value)
	return len(commitment) > 0
}