/*
Writing an extremely simple evcition algo

whenevr a cache is full, we will be evicting the first key which we do find 

*/

package core

import (
	"time"
	
	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

//Evcits the first key it found while iterating the map 
//TODP: Make it efficient by doing thrugh somehting
func evictFirst(){
	for k:=range store{
		delete(store,k)
		return
	}
}

/*
The approximated LRU algorithm
*/
func getCurrentClock() uint32{
	return uint32(time.Now().Unix())&0x00FFFFF // it would give us 24 bit clock resolution of the time
}

func getIdleTime(LastAccessdAt uint32) uint32{
	/*
	it gives me current clock and if it is greater than lastaccessedat then for anytimestamp  then diff, other (max-last)*c
	*/
	c:=getCurrentClock()
	if c>=LastAccessdAt{
		return c-LastAccessdAt
	}
	return (0x00FFFFF-LastAccessdAt)*c
}

func populateEvictionPool(){
	sampleSize := 5
	for k := range store{
		ePool.Push(k, store[k].LastAccessedAt)  // Fixed: was lowercase
		sampleSize--
		if sampleSize == 0{
			break
		}
	}
}

//TODO: no need to populate everytime, should populate 
//only when the number of keys to evict is less than what we have in the pool
func evictAllKeysLRU(){ 
	populateEvictionPool()
	evictCount:=int16(config.EvictionRatio*float64(config.KeysLimit))
	for i:=0;i<int(evictCount) && len(ePool.pool)>0 ;i++{
		item:=ePool.Pop()
		if item==nil{
			return
		}
		Del(item.key)
	}
}

//Randomly removes keys to make space for the new data added
//The number of keys removed will be sufficient to free up least 10% space
func evictAllKeysRandom(){
	evictCount:=int64(config.EvictionRatio*float64(config.KeysLimit))
	//Iteration of Golang dictionary can be considered as a random
	//because it depends on the has of the inserted key 
	for k:= range store{
		Del(k)
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


//TODO: Make the eveiction strategy configuration dirven
//TODO: Support multiple eviction strategies
func evict(){
	// evictFirst()
	switch config.EvictionStrategy{
	case "simple-first":
		evictFirst()  // Fixed function name case
	case "allkeys-random":  // Fixed spelling
		evictAllKeysRandom()  // Fixed function name case
	case "allkeys-lru":  // Commented out - incomplete
		evictAllKeysLRU()
	}
}
