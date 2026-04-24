package core

func Shutdown(){
	//this function internally invokes BGREWRITEAOF, it would take the in-memory hash table and dump in it AOF File(Append only file)
	evalBGREWRITEAOF([]string{})
}