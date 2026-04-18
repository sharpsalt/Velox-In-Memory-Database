package core

import "strconv"


//Similar to
//tryObjectEncoding Function in redis
func deduceTypeEncoding(v string)(uint8,uint8){
	oType:=OBJ_TYPE_STRING
	if _,rr:=strconv.ParseInt(v,10,64);err==nil{
		return oType,OBJ_ENCODING_INT //we are converting ans stroing int as a string inside my redis object
	}
	if len(v)<=44{
		return oType,OBJ_ENCODING_EMBSTR //Object encoding is embedded string
	}
	return oType,OBJ_ENCODING_RAW
}

