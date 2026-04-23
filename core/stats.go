package core

var KeyspaceStat [4]map[string]int //just all global object
//eg i support 4 databases within my redis
//in redis you can have 16 databases in redis itself, by deault it goes from db0,db1,db2,...db15

func init(){
	// Initialize the maps for each database
	for i := 0; i < 4; i++ {
		KeyspaceStat[i] = make(map[string]int)
	}
}

//for which db,which metric,which value
func UpdatDBStat(num int, metric string, value int){
	// Check bounds before accessing
	if num >= 0 && num < len(KeyspaceStat) {
		KeyspaceStat[num][metric] = value
	}
}
