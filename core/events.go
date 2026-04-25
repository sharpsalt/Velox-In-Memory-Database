package core

import "log"

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