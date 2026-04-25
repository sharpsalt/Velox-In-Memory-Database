package core 


//TODO: Change ExpiresAt it to LRU Bits as handled by Redis
type Obj struct{
	TypeEncoding uint8
	Value        interface{} //which means we acn put literally anything,anyvalue and which would work fine
	// ExpiresAt    int64
	/*
	Earlier we use to have expireAt here, we don't need as we need LastAccessdAt
	since golang doesn't support bitfield so LastAccessedAt is storing everytime the key is getting accessed
	*/

	/*
	Redis allot 24 bits to these bits, but we will use 32 bits because
	golang does not support bitfields and we need not make this super-compplicated 
	by merging TypeEncoding + LastAccessedAt in one 32 bit integer
	but nonthelss, we can benchmarks and see how that fares
	For noe, we continue with 32 bit integer to stre th LastAccessedAt
	*/
	LastAccessedAt uint32
}
/*
C and c++ give us ways to apply bit fields on set, means in each set i can assign a fix number of bits
so i am deifning type encoding 


if i am storing type encoding in c/c++, i could do it via bit field s since i don't have assembler here so 
we are doing like this way, or or krke krrhe hai
*/

var OBJ_TYPE_STRING uint8=0<<4 //because first 4 bits i want to set
var OBJ_TYPE_HASH   uint8=1<<4 //hash type
var OBJ_TYPE_LIST   uint8=2<<4 //list type
var OBJ_TYPE_SET    uint8=3<<4 //set type

var OBJ_ENCODING_RAW      uint8=0
var OBJ_ENCODING_INT      uint8=1
var OBJ_ENCODING_ZIPLIST   uint8=2 //compact sequential storage for small hashes/lists
var OBJ_ENCODING_HT        uint8=3 //hashtable encoding (map[string]string for hashes, map[string]struct{} for sets)
var OBJ_ENCODING_QUICKLIST  uint8=4 //linked list of ziplists (used for lists)
var OBJ_ENCODING_INTSET     uint8=5 //sorted integer array (used for small integer-only sets)
var OBJ_ENCODING_EMBSTR uint8=8 //Object encoding is embedded string



