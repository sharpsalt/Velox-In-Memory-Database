package core

var KeyspaceStat [4]map[string]int //just all global object
//eg i support 4 databases within my redis
//in redis you can have 16 databases in redis itself, by deault it goes from db0,db1,db2,...db15

//for which db,which metric,which value
func UpdatDBStat(num int,metric string,value int){
	KeyspaceStat[num][metric]=value
}
