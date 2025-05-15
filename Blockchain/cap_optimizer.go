package blockchain

import (
	"sync"
	"time"
)

// VectorClock tracks causal relationships between events
type VectorClock map[string]int

func (vc VectorClock) Increment(nodeID string) {
	vc[nodeID]++
}

func (vc VectorClock) Merge(other VectorClock) VectorClock {
	result := make(VectorClock)
	for k, v := range vc {
		result[k] = v
	}
	for k, v := range other {
		if v > result[k] {
			result[k] = v
		}
	}
	return result
}

func (vc VectorClock) Entropy() int {
	sum := 0
	for _, v := range vc {
		sum += v
	}
	return sum
}

// NetworkStats tracks network health metrics
type NetworkStats struct {
	Latency        time.Duration
	PartitionProb  float64
	LastHeartbeats map[string]time.Time
	mutex          sync.RWMutex
}

func NewNetworkStats() *NetworkStats {
	return &NetworkStats{
		LastHeartbeats: make(map[string]time.Time),
	}
}

func (ns *NetworkStats) UpdateNodeHeartbeat(nodeID string) {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()
	ns.LastHeartbeats[nodeID] = time.Now()
}

func (ns *NetworkStats) PredictPartitionProbability() float64 {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	
	now := time.Now()
	timeoutNodes := 0
	
	for _, lastBeat := range ns.LastHeartbeats {
		if now.Sub(lastBeat) > 5*time.Second {
			timeoutNodes++
		}
	}
	
	probFromTimeouts := float64(timeoutNodes) / float64(len(ns.LastHeartbeats)+1)
	probFromLatency := float64(ns.Latency) / float64(time.Second)
	
	ns.PartitionProb = (probFromTimeouts + probFromLatency) / 2
	if ns.PartitionProb > 1.0 {
		ns.PartitionProb = 1.0
	}
	
	return ns.PartitionProb
}

// ConsistencyLevel defines different consistency guarantees
type ConsistencyLevel int

const (
	EventualConsistency ConsistencyLevel = 1
	CausalConsistency   ConsistencyLevel = 2
	StrongConsistency   ConsistencyLevel = 3
)

// ConsistencyOrchestrator dynamically adjusts consistency based on network conditions
type ConsistencyOrchestrator struct {
	NodeConsistency map[string]ConsistencyLevel
	NetworkStats    *NetworkStats
	mutex           sync.RWMutex
}

func NewConsistencyOrchestrator() *ConsistencyOrchestrator {
	return &ConsistencyOrchestrator{
		NodeConsistency: make(map[string]ConsistencyLevel),
		NetworkStats:    NewNetworkStats(),
	}
}

func (co *ConsistencyOrchestrator) DynamicAdjustConsistency(nodeID string) ConsistencyLevel {
	co.mutex.Lock()
	defer co.mutex.Unlock()
	
	partitionProb := co.NetworkStats.PredictPartitionProbability()
	
	var level ConsistencyLevel
	
	if partitionProb > 0.7 {
		level = EventualConsistency      // High risk of partitions, prioritize availability
	} else if partitionProb > 0.3 {
		level = CausalConsistency        // Moderate risk, balanced approach
	} else {
		level = StrongConsistency        // Low risk, prioritize consistency
	}
	
	co.NodeConsistency[nodeID] = level
	return level
}

func (co *ConsistencyOrchestrator) GetConsistencyLevel(nodeID string) ConsistencyLevel {
	co.mutex.RLock()
	defer co.mutex.RUnlock()
	
	level, exists := co.NodeConsistency[nodeID]
	if !exists {
		return CausalConsistency // Default to middle ground
	}
	return level
}

// ConflictResolver handles state conflicts using vector clocks
type ConflictResolver struct{}

func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

func (cr *ConflictResolver) ResolveConflicts(values []string, clocks []VectorClock) string {
	if len(values) == 0 || len(clocks) == 0 {
		return ""
	}
	
	if len(values) == 1 {
		return values[0]
	}
	
	// Check for causal dominance
	dominantIdx := -1
	
	for i := 0; i < len(clocks); i++ {
		isDominant := true
		
		for j := 0; j < len(clocks); j++ {
			if i == j {
				continue
			}
			
			dominates := true
			for k, v := range clocks[j] {
				if clocks[i][k] < v {
					dominates = false
					break
				}
			}
			
			if !dominates {
				isDominant = false
				break
			}
		}
		
		if isDominant {
			dominantIdx = i
			break
		}
	}
	
	if dominantIdx >= 0 {
		return values[dominantIdx]
	}
	
	// If no dominant version, use entropy-based resolution
	minEntropy := int(^uint(0) >> 1) // Max int
	chosenIdx := 0
	
	for i, vc := range clocks {
		entropy := vc.Entropy()
		if entropy < minEntropy {
			minEntropy = entropy
			chosenIdx = i
		}
	}
	
	return values[chosenIdx]
}

// AdaptiveTimeoutManager handles dynamic timeouts based on network conditions
type AdaptiveTimeoutManager struct {
	baseTimeout time.Duration
	maxTimeout  time.Duration
	backoffRate float64
	netStats    *NetworkStats
}

func NewAdaptiveTimeoutManager(netStats *NetworkStats) *AdaptiveTimeoutManager {
	return &AdaptiveTimeoutManager{
		baseTimeout: 100 * time.Millisecond,
		maxTimeout:  10 * time.Second,
		backoffRate: 1.5,
		netStats:    netStats,
	}
}

func (atm *AdaptiveTimeoutManager) CalculateTimeout() time.Duration {
	partitionProb := atm.netStats.PredictPartitionProbability()
	
	adjustedTimeout := atm.baseTimeout
	
	if partitionProb > 0.5 {
		factor := 1 + (partitionProb * atm.backoffRate)
		adjustedTimeout = time.Duration(float64(atm.baseTimeout) * factor)
	}
	
	if adjustedTimeout > atm.maxTimeout {
		adjustedTimeout = atm.maxTimeout
	}
	
	return adjustedTimeout
}

func (co *ConsistencyOrchestrator) SetConsistency(level ConsistencyLevel) {
    co.mutex.Lock()
    defer co.mutex.Unlock()
    for node := range co.NodeConsistency {
        co.NodeConsistency[node] = level
    }
}