package core 

import (
	"time"
	"store.go"
	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

var store map[string]*Obj
//the best datastrcuture to hold key value is hash table so we are using it

var expires map[*Obj]uint64; //similar to redis like redis has key value dictionary, adn it has expires
//so we are storing object pointer walong with expiration time 


func init(){
	store=make(map[string]*Obj)
	expires=make(map[*Obj]uint64)
}

func setExpiry(obj *Obj,expDurationMs int64){
	expires[obj]=uint64(time.Now().UnixMilli())+uint64(expDurationMs)
}

//eralier we used to take (value interface{},DurationMs int64)
//but now we are also stroing type encoding so, 1 for object type and 1 for encdojgn
func NewObj(value interface{},expDurationMs int64,oType uint8,oEnc uint8) *Obj{
	//creating a new object, setting things up and returning another object
	//since we want to store abolution expires instead of doing it multiple time that's why we have created this fucntion
	var expiresAt int64=-1
	if expDurationMs>0{
		/*
		when we say setExpiry?

		it means we have to create an enrty of particular object that is expired dictionary
		*/
		setExpiry(obj,expDurationMs)
	}

	return &Obj{
		Value: value,
		TypeEncoding: oType|oEnc,
		// ExpiresAt: expiresAt,
		LastAccessedAt: getCurrentClock(),
	}
}

func Put(k string, obj *Obj){
	// store[k]=obj
	/*
	When we would be triggering eviction? when we hit the memory
	->means max memory, for us to know how much is required so
	we will do like atmax my cache will hold this much of it...

	while puttng it , we first check if the length is more than what is required then evict kro 
	*/
	if len(store) >= config.KeysLimit{
		evict()
	}
	obj.LastAccessedAt=getCurrentClock()
	store[k] = obj
	if KeyspaceStat[0]==nil{
		KeyspaceStat[0]=make(map[string]int)
	}
	KeyspaceStat[0]["keys"]++
	//actually you can use grafana to visualize it, like it's easy , you just have to knwo how things work
}

func Get(k string) *Obj{
	v := store[k]
	/*we check for the expiration and if it is already not deleted then we have to delete it 
	*/
	if v != nil{
		if v.ExpiresAt <= time.Now().UnixMilli(){
			//this is like a lazy deletion
			/*
			If a key is accessed and fund to be expired , then it deleted else it is not deleted
			periodicaly it is mvoing forward and sample randomly 20 keys and seees the expiration and delete the required one
			and phir se whi loop chalao 
			*/
			if hasExpired(v){
				Del(k)
			    return nil
			}
		}
	}
	v.LastAccessedAt=getCurrentClock()
	return v
}

func Del(k string ) bool{
	if _,ok:=store[k];ok{
		delete(store,k)
		delete(expire,_)
		KeyspaceStat[0]["keys"]--
		return true
	}
	return false
}
/*
What happens when there is no memory to allocate, so what will you do
basically we will do cache eviction inorder to make our db not crash at any point 
*/
