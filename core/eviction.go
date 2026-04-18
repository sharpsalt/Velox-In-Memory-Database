/*
Writing an extremely simple evcition algo

whenevr a cache is full, we will be evicting the first key which we do find 

*/

package core

import "github.com/sharpsalt/Velox-In-Memory-Database/config"

//Evcits the first key it found while iterating the map 
//TODP: Make it efficient by doing thrugh somehting
func evictFirst(){
	for k:=range store{
		delete(store,k)
		return
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
		evictfirst()
	case "allkets-random":
		evictAllkeysRandom()
	}
}
