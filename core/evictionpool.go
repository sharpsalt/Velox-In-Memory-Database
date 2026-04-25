package core

import (
	"container/heap"
)

/*
Eviction Pool — a max-heap of eviction candidates ordered by idle time.

This is the heart of the approximated LRU algorithm:
- The pool has a fixed max size (16 by default, same as Redis)
- It accumulates the best eviction candidates seen across multiple sampling rounds
- Items with the HIGHEST idle time (most idle = least recently used) are at the top
- When eviction is needed, we Pop from the top to get the best candidate

Why a heap?
- Push: O(log n) — way better than sort.Sort's O(n log n) on every insert
- Pop:  O(log n) — same
- Fix:  O(log n) — for updating when we replace the worst candidate

Why a pool instead of just sampling?
- Without a pool, each eviction samples 5 keys and picks the worst — might miss actually idle keys
- With a pool, candidates from PREVIOUS samples persist — the pool converges to holding
  the truly least-recently-used keys over time, even though each individual sample is small
- Redis proved this approach: with pool size 16 + sample size 5, you get >90% of true LRU accuracy
*/

type PoolItem struct{
	key string
	lastaccessedat uint32
	index int // index in the heap — maintained by heap.Interface methods
}

type EvictionPool struct{
	pool []*PoolItem   //it is basically an array of pool items 
	keyset map[string]*PoolItem  //keyset tracks which keys are already in the pool (O(1) lookup)
}

// ---- heap.Interface implementation ----
// The heap is a MAX-HEAP by idle time: items with highest idle time bubble up

func (pq *EvictionPool) Len() int {
	return len(pq.pool)
}

func (pq *EvictionPool) Less(i, j int) bool {
	// MAX-HEAP: item with MORE idle time should be at the top (index 0)
	// so we pop the least-recently-used item first
	return getIdleTime(pq.pool[i].lastaccessedat) > getIdleTime(pq.pool[j].lastaccessedat)
}

func (pq *EvictionPool) Swap(i, j int) {
	pq.pool[i], pq.pool[j] = pq.pool[j], pq.pool[i]
	pq.pool[i].index = i
	pq.pool[j].index = j
}

// Push is called by heap.Push — adds element to the backing array
func (pq *EvictionPool) Push(x interface{}) {
	item := x.(*PoolItem)
	item.index = len(pq.pool)
	pq.pool = append(pq.pool, item)
}

// Pop is called by heap.Pop — removes element from the backing array
func (pq *EvictionPool) Pop() interface{} {
	old := pq.pool
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak (don't hold reference to popped item)
	item.index = -1
	pq.pool = old[0 : n-1]
	return item
}

// ---- Public API ----

// PushItem adds a key to the eviction pool.
//
// How it works:
// 1. If key is already in pool → skip (no duplicates)
// 2. If pool has room → just add it, O(log n) heap insert
// 3. If pool is full → compare with the LEAST idle item in pool
//    - If new item is more idle → replace the least idle item
//    - This ensures pool always holds the BEST eviction candidates
//
// Finding the least idle item in a max-heap:
// In a max-heap, the minimum is among the leaves (bottom half of the array).
// We scan the bottom half to find the true minimum — still O(n/2) but n=16 so it's constant.
func (pq *EvictionPool) PushItem(key string, lastaccessedat uint32) {
	// Skip if already in pool
	if _, ok := pq.keyset[key]; ok {
		return
	}

	newIdleTime := getIdleTime(lastaccessedat)
	item := &PoolItem{key: key, lastaccessedat: lastaccessedat}

	if len(pq.pool) < ePoolSizeMax {
		// Pool has room — just add
		pq.keyset[key] = item
		heap.Push(pq, item)
	} else {
		// Pool is full — find the item with LEAST idle time (worst eviction candidate)
		// In a max-heap, minimum is among the leaves: indices [n/2, n)
		minIdx := len(pq.pool) / 2
		for i := minIdx + 1; i < len(pq.pool); i++ {
			if getIdleTime(pq.pool[i].lastaccessedat) < getIdleTime(pq.pool[minIdx].lastaccessedat) {
				minIdx = i
			}
		}

		// Only replace if new item is a better candidate (more idle)
		if newIdleTime > getIdleTime(pq.pool[minIdx].lastaccessedat) {
			// Remove old item from keyset
			delete(pq.keyset, pq.pool[minIdx].key)
			// Overwrite in-place and fix heap position
			pq.pool[minIdx].key = key
			pq.pool[minIdx].lastaccessedat = lastaccessedat
			pq.keyset[key] = pq.pool[minIdx]
			heap.Fix(pq, minIdx) // O(log n)
		}
	}
}

// PopItem removes and returns the item with the HIGHEST idle time
// (the best eviction candidate — least recently used)
func (pq *EvictionPool) PopItem() *PoolItem {
	if len(pq.pool) == 0 {
		return nil
	}
	item := heap.Pop(pq).(*PoolItem)
	delete(pq.keyset, item.key)
	return item
}

// PoolStats returns diagnostic info about the eviction pool
// Used by evalLRU for the LRU stats command
func (pq *EvictionPool) PoolStats() (size int, topIdleTime uint32) {
	size = len(pq.pool)
	if size > 0 {
		topIdleTime = getIdleTime(pq.pool[0].lastaccessedat)
	}
	return
}

func newEvictionPool() *EvictionPool {
	ep := &EvictionPool{
		pool:   make([]*PoolItem, 0, ePoolSizeMax), // pre-allocate capacity
		keyset: make(map[string]*PoolItem),
	}
	heap.Init(ep)
	return ep
}

// ePoolSizeMax is the maximum number of eviction candidates to keep in the pool
// Redis uses 16, which gives excellent accuracy with minimal memory overhead
var ePoolSizeMax int=16
var ePool *EvictionPool=newEvictionPool()
