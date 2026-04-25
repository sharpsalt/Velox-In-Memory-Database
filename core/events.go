package core

import (
	"log"
	"time"
)

// GlobalCachedTime holds the current time in Unix milliseconds.
// It is updated once per event loop tick to eliminate thousands of expensive
// time.Now().UnixMilli() system calls per second during high QPS loads.
var GlobalCachedTime int64

func init() {
	GlobalCachedTime = time.Now().UnixMilli()
}

func UpdateCachedTime() {
	GlobalCachedTime = time.Now().UnixMilli()
}

// InitStore replays the AOF file on startup to restore persisted data
// Call this once before accepting any client connections
func InitStore() {
	log.Println("Initializing store — replaying AOF if available...")
	ReplayAOF()
	log.Println("Store initialized, total keys:", StoreLen())
}

func Shutdown(){
	//this function internally invokes BGREWRITEAOF, it would take the in-memory hash table and dump in it AOF File(Append only file)
	log.Println("Shutting down — dumping AOF...")
	evalBGREWRITEAOF([]string{})
	log.Println("Shutdown complete")
}