/*
Writing an extremely simple evcition algo

whenevr a cache is full, we will be evicting the first key which we do find 

*/

package core

//Evcits the first key it found while iterating the map 
//TODP: Make it efficient by doing thrugh somehting
func evictFirst(){
	for k:=range store{
		delete(store,k)
		return
	}
}


//TODO: Make the eveiction strategy configuration dirven
//TODO: Support multiple eviction strategies
func evict(){
	evictFirst()
}
