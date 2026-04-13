package core 

import "time"

var store map[string]*Obj
//the best datastrcuture to hold key value is hash table so we are using it

type Obj struct{
	Value interface{} //which means we acn put literally anything,anyvalue and which would work fine
	ExpiresAt int64 //is time ke baad expire hojayega wo
}

func init(){
	store=make(map[string]*Obj)
}

func NewObj(value interface{},DurationMs int64) *Obj{
	//creating a new object, setting things up and returning another object
	//since we want to store abolution expires instead of doing it multiple time that's why we have created this fucntion
	var expiresAt int64=-1
	if DurationMs>0{
		expiresAt=time.Now().UnixMilli()+DurationMs
	}

	return &Obj{
		Value: value,
		ExpiresAt: expiresAt
	}
}

func Put(k string,obj *Obj){
	// store[k]=obj
	/*
	When we would be triggering eviction? when we hit the memory
	->means max memory, for us to know how much is required so
	we will do like atmax my cache will hold this much of it...

	while puttng it , we first check if the length is more than what is required then evict kro 
	*/
	if len(store)>=config.KeysLimit{
		evict()
	}
	store[k]=obj
}

func Get(k string) *Obj{
    v:=store[k]
	/*we check for the expiration and if it is already not deleted then we have to delete it 
	*/
	if v!=nil{
		if v.ExpiresAt<=time.Now().UnixMilli(){
			//this is like a lazy deletion
			/*
			If a key is accessed and fund to be expired , then it deleted else it is not deleted
			periodicaly it is mvoing forward and sample randomly 20 keys and seees the expiration and delete the required one
			and phir se whi loop chalao 
			*/
			delete(store,k)
			return nil
		}
	}
	return v
}

func Del(k string ) bool{
	if _,ok:=store[k];ok{
		delete(store,k)
		return true
	}
	return false
}
/*
What happens when there is no memory to allocate, so what will you do
basically we will do cache eviction inorder to make our db not crash at any point 
*/
