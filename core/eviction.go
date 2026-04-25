/*
Approximated LRU Eviction — modeled after Redis

How Redis does LRU eviction (and how we do it here):

1. TRUE LRU requires maintaining an ordered data structure (e.g., doubly linked list)
   tracking every key by access time. This is O(1) per access but costs significant memory
   (two pointers per key = 16 bytes overhead on 64-bit systems).

2. APPROXIMATED LRU (what Redis and Velox use):
   - Each object stores a small LRU clock (24 bits in Redis, 32 bits here)
   - When eviction is needed, we DON'T scan all keys
   - Instead, we SAMPLE N random keys (configurable via LRUSampleSize, default 5)
   - We maintain an "Eviction Pool" of the 16 best eviction candidates seen so far
   - Each sampling round, sampled keys compete with pool entries — only the best survive
   - When we need to evict, we pop from the pool (highest idle time = least recently used)

   With just 5 samples per round, this gets ~90% of true LRU accuracy.
   With 10 samples, it's nearly indistinguishable from true LRU.

   The key insight: Go's map iteration IS randomized, so iterating N keys from the map
   gives us a random sample for free — no need for a separate random index structure.

3. The pool PERSISTS across eviction rounds — it accumulates the best candidates over time.
   This means even with small samples, the pool converges to holding the truly idle keys.
*/

package core

import (
	"log"
	"time"
	
	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

//Evcits the first key it found while iterating the map 
//TODP: Make it efficient by doing thrugh somehting
// NOTE: caller must hold storeMu.Lock()
func evictFirst(){
	for k:=range store{
		delLocked(k)
		return
	}
}

/*
The approximated LRU clock
We use a 20-bit clock with second resolution
This gives us ~12 days before the clock wraps around
Redis uses 24 bits giving ~194 days, but 20 bits is fine for our use case
*/
func getCurrentClock() uint32{
	return uint32(time.Now().Unix())&0x00FFFFF // 20-bit clock resolution
}

// getIdleTime calculates how long ago a key was last accessed
// Handles clock wraparound correctly (when current time < last access due to overflow)
func getIdleTime(lastAccessedAt uint32) uint32{
	c:=getCurrentClock()
	if c>=lastAccessedAt{
		return c-lastAccessedAt
	}
	// Clock wrapped around: total idle = (max - last) + current
	return (0x00FFFFF-lastAccessedAt)+c
}

// GetKeyIdleTime returns the idle time in seconds for a given key
// This is used by the OBJECT IDLETIME command
// Returns idle time and whether the key exists
func GetKeyIdleTime(key string) (uint32, bool) {
	storeMu.RLock()
	defer storeMu.RUnlock()
	obj, exists := store[key]
	if !exists {
		return 0, false
	}
	return getIdleTime(obj.LastAccessedAt), true
}

/*
populateEvictionPool samples N random keys from the store and inserts them into
the eviction pool. Go map iteration is randomized, so iterating gives us a free
random sample.

The pool persists across calls — it accumulates the best eviction candidates
over multiple sampling rounds. This is what makes approximated LRU accurate:
even though each sample is small, the pool remembers the best candidates from
ALL previous samples.

NOTE: caller must hold storeMu.Lock()
*/
func populateEvictionPool(){
	sampleSize := config.LRUSampleSize
	sampled := 0
	for k := range store{
		obj := store[k]
		if obj == nil {
			continue
		}
		// PushItem will only add to pool if the key is a better eviction candidate
		// than what's already there (or if there's room)
		ePool.PushItem(k, obj.LastAccessedAt)
		sampled++
		if sampled >= sampleSize{
			break
		}
	}
}

/*
evictAllKeysLRU is the main approximated LRU eviction function.

Algorithm:
1. Sample LRUSampleSize random keys from the store
2. Insert them into the eviction pool (pool keeps the best candidates)
3. Pop from the pool (highest idle time first) and delete those keys
4. Repeat sampling if pool runs dry but we still need to evict more

NOTE: caller must hold storeMu.Lock()
*/
func evictAllKeysLRU(){ 
	evictCount:=int(config.EvictionRatio*float64(config.KeysLimit))
	if evictCount < 1 {
		evictCount = 1
	}

	evicted := 0
	for evicted < evictCount {
		// Populate the pool with fresh samples
		populateEvictionPool()

		// If pool is empty even after sampling, nothing left to evict
		if ePool.Len() == 0 {
			break
		}

		// Pop the best eviction candidate (highest idle time)
		item := ePool.PopItem()
		if item == nil {
			break
		}

		// Verify the key still exists (it may have been deleted by another path)
		if _, exists := store[item.key]; exists {
			delLocked(item.key)
			evicted++
		}
	}

	if evicted > 0 {
		log.Printf("approximated LRU: evicted %d keys (pool size: %d, sample size: %d)\n",
			evicted, ePool.Len(), config.LRUSampleSize)
	}
}

//Randomly removes keys to make space for the new data added
//The number of keys removed will be sufficient to free up least 10% space
// NOTE: caller must hold storeMu.Lock()
func evictAllKeysRandom(){
	evictCount:=int64(config.EvictionRatio*float64(config.KeysLimit))
	//Iteration of Golang dictionary can be considered as a random
	//because it depends on the has of the inserted key 
	for k:= range store{
		delLocked(k)
		evictCount--
		if evictCount==0{
			break
		}
	}
}
/*
How do we know that our eviction is working correctly??

which means when we are doing large number of key sets , we are putting lots of keys in db , let's say we put kelimit to 100
, how do we sure that we are not breachng it , so every dbs supports statistics
*/


// evict is called when the store is full (len(store) >= KeysLimit)
// It dispatches to the configured eviction strategy.
// NOTE: caller must hold storeMu.Lock()
func evict(){
	switch config.EvictionStrategy{
	case "simple-first":
		evictFirst()
	case "allkeys-random":
		evictAllKeysRandom()
	case "allkeys-lru":
		evictAllKeysLRU()
	default:
		// Fallback to LRU if unknown strategy
		evictAllKeysLRU()
	}
}
