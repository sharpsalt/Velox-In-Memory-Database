package core 


//TODO: Change ExpiresAt it to LRU Bits as handled by Redis
type Obj struct{
	TypeEncoding uint8
	Value        interface{}
	ExpiresAt    int64
}
/*
C and c++ give us ways to apply bit fields on set, means in each set i can assign a fix number of bits
so i am deifning type encoding 


if i am storing type encoding in c/c++, i could do it via bit field s since i don't have assembler here so 
we are doing like this way, or or krke krrhe hai
*/

var OBJ_TYPE_STRING uint8=0<<4 //because first 4 bits i want to set

var OBJ_ENCODING_RAW uint8=0
var OBJ_ENCODING_INT uint8=1
var OBJ_ENCODING_ENDSTR uint8=8



