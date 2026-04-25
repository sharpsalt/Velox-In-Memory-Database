package core 

import (
	"sync"
	
	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

var store map[string]*Obj
//the best datastrcuture to hold key value is hash table so we are using it

var expires map[*Obj]uint64; //similar to redis like redis has key value dictionary, adn it has expires
//so we are storing object pointer walong with expiration time 

// storeMu protects both the store and expires maps from concurrent access
// Use RLock/RUnlock for read-only operations (Get, iteration)
// Use Lock/Unlock for write operations (Put, Del, setExpiry)
var storeMu sync.RWMutex


func init(){
	store=make(map[string]*Obj)
	expires=make(map[*Obj]uint64)
}

func setExpiry(obj *Obj,expDurationMs int64){
	// NOTE: caller must hold storeMu.Lock() before calling this
	// PERF: using GlobalCachedTime instead of time.Now().UnixMilli() to avoid syscall
	expires[obj]=uint64(GlobalCachedTime)+uint64(expDurationMs)
}

var objPool = sync.Pool{
	New: func() interface{} {
		return &Obj{}
	},
}

//eralier we used to take (value interface{},DurationMs int64)
//but now we are also stroing type encoding so, 1 for object type and 1 for encdojgn
func NewObj(value interface{},expDurationMs int64,oType uint8,oEnc uint8) *Obj{
	//creating a new object, setting things up and returning another object
	//since we want to store abolution expires instead of doing it multiple time that's why we have created this fucntion
	// var expiresAt int64=-1

	/*
		when we say setExpiry?

		it means we have to create an enrty of particular object that is expired dictionary
		*/
	obj := objPool.Get().(*Obj)
	obj.Value = value
	obj.TypeEncoding = oType | oEnc
	obj.LastAccessedAt = getCurrentClock()
    // NOTE: setExpiry is called inside Put which holds the lock,
    // or during NewObj before the object is visible to other goroutines.
    // For safety, we don't set expiry here — it's done in Put after acquiring the lock.
    // We store the expDurationMs in a temporary way and let Put handle it.
    if expDurationMs>0{
       setExpiry(obj,expDurationMs)
    }
    return obj
}

func Put(k string, obj *Obj){
	// store[k]=obj
	/*
	When we would be triggering eviction? when we hit the memory
	->means max memory, for us to know how much is required so
	we will do like atmax my cache will hold this much of it...

	while puttng it , we first check if the length is more than what is required then evict kro 
	*/
	storeMu.Lock()
	defer storeMu.Unlock()

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
	storeMu.RLock()
	v := store[k]
	/*we check for the expiration and if it is already not deleted then we have to delete it 
	*/
	if v != nil{
		// if v.ExpiresAt <= time.Now().UnixMilli(){
			//this is like a lazy deletion
			/*
			If a key is accessed and fund to be expired , then it deleted else it is not deleted
			periodicaly it is mvoing forward and sample randomly 20 keys and seees the expiration and delete the required one
			and phir se whi loop chalao 
			*/
			if hasExpired(v){
				storeMu.RUnlock()
				// Upgrade to write lock for deletion
				storeMu.Lock()
				// Re-check after acquiring write lock (another goroutine may have deleted it)
				if _, exists := store[k]; exists {
					delLocked(k)
				}
				storeMu.Unlock()
			    return nil
			}
		// }
	}
	if v!=nil{
		v.LastAccessedAt=getCurrentClock()
	}
	storeMu.RUnlock()
	return v
}

func Del(k string) bool{
	storeMu.Lock()
	defer storeMu.Unlock()
	return delLocked(k)
}

// delLocked performs deletion while the caller already holds storeMu.Lock()
// This avoids double-locking when called from Get() or eviction functions
func delLocked(k string) bool{
	obj, ok := store[k]
	if ok {
		delete(store, k)
		delete(expires, obj)
		KeyspaceStat[0]["keys"]--
		
		// PERF: Clean up fields to avoid memory leaks before pooling
		obj.Value = nil
		objPool.Put(obj)
		
		return true
	}
	return false
}

// StoreLen returns the current number of keys (thread-safe)
func StoreLen() int {
	storeMu.RLock()
	defer storeMu.RUnlock()
	return len(store)
}

/*
What happens when there is no memory to allocate, so what will you do
basically we will do cache eviction inorder to make our db not crash at any point 
*/
