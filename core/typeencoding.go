package core
import "errors"

/*
From a particular type encoding object if you want to get type of it
type is definately the first 4 bits 

so inorder to extract that i am right shifting by 4 and then left shift by 4
*/
func getType(te uint8) uint8{
	return (te>>4)<<4
}

func getEncoding(te uint8) uint8{
	return te & 0b00001111
}

func assertType(te uint8,t uint8)error{
	if getType(te)!=t{
		return errors.New("the operation is not permitted on this type")
	}
	return nil
}


func assertEncoding(te uint8,e uint8)error{
	if getEncoding(te)!=e{
		return errors.New("the operations is not permitted on this encoding")
	}
	return nil
}

// TypeName returns the human-readable type name for the OBJECT TYPE command
func TypeName(te uint8) string {
	switch getType(te) {
	case OBJ_TYPE_STRING:
		return "string"
	case OBJ_TYPE_HASH:
		return "hash"
	case OBJ_TYPE_LIST:
		return "list"
	case OBJ_TYPE_SET:
		return "set"
	case OBJ_TYPE_ZSET:
		return "zset"
	default:
		return "none"
	}
}

// EncodingName returns the human-readable encoding name for the OBJECT ENCODING command
func EncodingName(te uint8) string {
	switch getEncoding(te) {
	case OBJ_ENCODING_RAW:
		return "raw"
	case OBJ_ENCODING_INT:
		return "int"
	case OBJ_ENCODING_ZIPLIST:
		return "ziplist"
	case OBJ_ENCODING_HT:
		return "hashtable"
	case OBJ_ENCODING_QUICKLIST:
		return "quicklist"
	case OBJ_ENCODING_INTSET:
		return "intset"
	case OBJ_ENCODING_SKIPLIST:
		return "skiplist"
	case OBJ_ENCODING_EMBSTR:
		return "embstr"
	default:
		return "unknown"
	}
}




























