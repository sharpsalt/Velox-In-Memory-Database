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
	store[k]=obj
}

func Get(k string) *Obj{
	return store[k]
}
