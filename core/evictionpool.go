package core

import{
	"sort"
}


type PoolItem struct{
	key string
	lastaccessedat uint32
}

//TODO: when last accessed at of object changes
//updates the poolItem corresponding to that
type EvictionPool struct{
	pool []*PoolItem   //it is basically an array of pool items 
	keyset map[string]*PoolItem  //na keyset , like the key which are present.
}

type ByIdleTime []*PoolItem

func (a ByIdleTime) len() int{
	return len(a)
}

func (a ByIdleTime) Swap(i int,j int){
	a[i],a[j]=a[j],a[i]
}

func (a ByIdleTime) Less(i int,j int) bool{//basically it is a comparator function which i am suing to sort the time
	return getIdleTime(a[i].lastaccessedat)>getIdleTime(a[j].lastaccessedat)
}

//TODO: Make the Implementation efficient to not need repeated sorting
func (pq *EvictionPool) Push(key string,lastaccessedat uint32){
	_,ok:=pq.keyset[key]
	if ok{
		//while pushing it into eviction pool if it already exists then we don;t have to push it again it in eveiction pool
		return
	}
	ietm:=&PoolItem(key:key,lastaccessedat:lastaccessedat)
	if len(pq.pool)<ePoolSizeMax{
		//which means there is some space for adding element
		//remeber eviction pool is needed to be sorted by idle time
		pq.keyset[key]=items
		pq.pool=append(pq.pool,item)


		//Performance bottleneck
		sort.Sort(ByIdleTime(pq.pool))
	}else if lastaccessedat>pq.pool[len(pq.pool)-1].lastaccessedat{
		//if i have no space in eviction pool but the element which i have smapled is worse than my current 
		//i will create space for that by removing the 1st one and adding new element by appending it 
		///so this way we are ensuring that our pool contains best possible candidates to be evcited
		pq.pool=pq.pool[1:]
		pq.keyset[key]=item
		pq.pool=append(pq.pool,item)
	}
}

func (pq *EvictionPool) Pop() *PoolItem{
	if len(pq.pool)==0{
		return nil
	}
	item:=pq.pool[0]
	pq.pool=pq.pool[1:]
	return item
}

func newEvictionPool(size int) *EvictionPool{
	return &EvictionPool{
		pool: make([]*PoolItem,size)
		keyset: make(map[string]*PoolItem)
	}
}

var ePoolSizeMax int=16
var ePool *EvictionPool=newEvictionPool(0)


