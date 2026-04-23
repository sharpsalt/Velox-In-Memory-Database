package core

import (
	"log"
	"time"
)

func hasExpired(obj *Obj) bool{
	exp,ok:=expires[obj]
	if !ok{
		return false
	}
	return exp<=uint64(time.Now().UnixMilli())
}

func getExpiry(obj *Obj)(uint64,bool){
	exp,ok:=expires[obj]
	return exp,ok
}

//Delete all the expired keys- the active way
//Sampling approach: https://redis.io/commands/expire/

//TODO: Optimize
//  -Sampling
//  -Unecessary iteration
func expireSample() float32{
	var limit int = 20
	var expiresCount int = 0

	//assuming iteration of golang hash table in randomized    
	for key, obj := range store{
		// if obj.ExpiresAt != -1{
		// 	limit--
		// 	//if the key is expired
		// 	if obj.ExpiresAt <= time.Now().UnixMilli(){
		// 		delete(store, key)
		// 		expiresCount++
		// 	}
		// }
		if hasExpired(obj){
			delete(store,key)
			expiresCount++
			limit--
		}

		//one we iterated to 20 keys that have some expirations set
		//we break the loop
		if limit == 0{
			break
		}
	}
	return float32(expiresCount)/float32(20.0)
}



func DeleteExpiredKey(){
	for{
		frac := expireSample()
		//if the sample had less than 25% keys required
		//we break the loop
		if frac < 0.25{
			break
		}
	}//a normal active deletion flow would happen like it 
	log.Println("deleted the expired but undeleted logs, total keys ", len(store))
}