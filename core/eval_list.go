/*
List Commands — LPUSH, RPUSH, LPOP, RPOP, LRANGE, LLEN, LINDEX

Lists use QuickList internally (OBJ_ENCODING_QUICKLIST).
*/
package core

import (
	"errors"
	"strconv"
)

func listGetOrCreate(key string) (*Obj, []byte) {
	obj := Get(key)
	if obj == nil {
		obj = NewObj(NewQuickList(), -1, OBJ_TYPE_LIST, OBJ_ENCODING_QUICKLIST)
		Put(key, obj)
		return obj, nil
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
		return nil, Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}
	return obj, nil
}

func evalLPUSH(args []string) []byte {
	if len(args) < 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'lpush' command"), false)
	}
	key := args[0]
	obj, errResp := listGetOrCreate(key)
	if errResp != nil {
		return errResp
	}
	ql := obj.Value.(*QuickList)
	for _, val := range args[1:] {
		ql.LPush(val)
	}
	return Encode(int64(ql.Len()), false)
}

func evalRPUSH(args []string) []byte {
	if len(args) < 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'rpush' command"), false)
	}
	key := args[0]
	obj, errResp := listGetOrCreate(key)
	if errResp != nil {
		return errResp
	}
	ql := obj.Value.(*QuickList)
	for _, val := range args[1:] {
		ql.RPush(val)
	}
	return Encode(int64(ql.Len()), false)
}

func evalLPOP(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'lpop' command"), false)
	}
	key := args[0]
	obj := Get(key)
	if obj == nil {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}
	ql := obj.Value.(*QuickList)
	val, ok := ql.LPop()
	if !ok {
		return RESP_NIL
	}
	if ql.Len() == 0 {
		Del(key)
	}
	return Encode(val, false)
}

func evalRPOP(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'rpop' command"), false)
	}
	key := args[0]
	obj := Get(key)
	if obj == nil {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}
	ql := obj.Value.(*QuickList)
	val, ok := ql.RPop()
	if !ok {
		return RESP_NIL
	}
	if ql.Len() == 0 {
		Del(key)
	}
	return Encode(val, false)
}

func evalLRANGE(args []string) []byte {
	if len(args) != 3 {
		return Encode(errors.New("ERR wrong number of arguments for 'lrange' command"), false)
	}
	key := args[0]
	start, err := strconv.Atoi(args[1])
	if err != nil {
		return Encode(errors.New("ERR value is not an integer or out of range"), false)
	}
	stop, err := strconv.Atoi(args[2])
	if err != nil {
		return Encode(errors.New("ERR value is not an integer or out of range"), false)
	}
	obj := Get(key)
	if obj == nil {
		return Encode([]string{}, false)
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}
	ql := obj.Value.(*QuickList)
	result := ql.Range(start, stop)
	return Encode(result, false)
}

func evalLLEN(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'llen' command"), false)
	}
	key := args[0]
	obj := Get(key)
	if obj == nil {
		return RESP_ZERO
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}
	ql := obj.Value.(*QuickList)
	return Encode(int64(ql.Len()), false)
}

func evalLINDEX(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'lindex' command"), false)
	}
	key := args[0]
	idx, err := strconv.Atoi(args[1])
	if err != nil {
		return Encode(errors.New("ERR value is not an integer or out of range"), false)
	}
	obj := Get(key)
	if obj == nil {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}
	ql := obj.Value.(*QuickList)
	val, ok := ql.Index(idx)
	if !ok {
		return RESP_NIL
	}
	return Encode(val, false)
}
