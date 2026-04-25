package core

import (
	"errors"
	"strconv"
)

/*
Sorted Set (ZSet) Commands — ZADD, ZSCORE, ZRANK, ZRANGE, ZCARD, ZREM

Internal Encoding:
We use a composite data structure for ZSet:
1. Skip List: For O(log N) rank operations and sorted range queries.
2. Hashtable (map): For O(1) score lookups given a member name.

This dual-structure approach matches Redis perfectly. Every insert/delete updates both.
Memory overhead is higher than standard sets, but performance is exceptional.
*/

// ZSet is the composite structure for Sorted Sets
type ZSet struct {
	sl   *SkipList
	dict map[string]float64
}

// newZSet creates a new empty ZSet
func newZSet() *ZSet {
	return &ZSet{
		sl:   CreateSkipList(),
		dict: make(map[string]float64),
	}
}

// zsetGetOrCreate retrieves the ZSet object or creates a new one
func zsetGetOrCreate(key string) (*Obj, []byte) {
	obj := Get(key)
	if obj == nil {
		zs := newZSet()
		obj = NewObj(zs, -1, OBJ_TYPE_ZSET, OBJ_ENCODING_SKIPLIST)
		Put(key, obj)
		return obj, nil
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return nil, Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}
	return obj, nil
}

// ZADD key score member [score member ...]
func evalZADD(args []string) []byte {
	if len(args) < 3 || len(args)%2 == 0 {
		return Encode(errors.New("ERR wrong number of arguments for 'zadd' command"), false)
	}

	key := args[0]
	obj, errResp := zsetGetOrCreate(key)
	if errResp != nil {
		return errResp
	}

	zs := obj.Value.(*ZSet)
	added := 0

	for i := 1; i < len(args); i += 2 {
		scoreStr, member := args[i], args[i+1]
		score, err := strconv.ParseFloat(scoreStr, 64)
		if err != nil {
			return Encode(errors.New("ERR value is not a valid float"), false)
		}

		// Check if member already exists
		if oldScore, exists := zs.dict[member]; exists {
			if oldScore != score {
				// Remove old node and insert new one
				zs.sl.Delete(oldScore, member)
				zs.sl.Insert(score, member)
				zs.dict[member] = score
			}
		} else {
			// Brand new member
			zs.sl.Insert(score, member)
			zs.dict[member] = score
			added++
		}
	}

	return Encode(int64(added), false)
}

// ZSCORE key member
func evalZSCORE(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'zscore' command"), false)
	}

	key, member := args[0], args[1]
	obj := Get(key)
	if obj == nil {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	zs := obj.Value.(*ZSet)
	if score, exists := zs.dict[member]; exists {
		// Format without trailing zeros, matching Redis
		scoreStr := strconv.FormatFloat(score, 'f', -1, 64)
		return Encode(scoreStr, false)
	}

	return RESP_NIL
}

// ZRANK key member
func evalZRANK(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'zrank' command"), false)
	}

	key, member := args[0], args[1]
	obj := Get(key)
	if obj == nil {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	zs := obj.Value.(*ZSet)
	score, exists := zs.dict[member]
	if !exists {
		return RESP_NIL
	}

	rank := zs.sl.GetRank(score, member)
	if rank == 0 {
		return RESP_NIL
	}
	
	// Redis rank is 0-based
	return Encode(int64(rank-1), false)
}

// ZRANGE key start stop
func evalZRANGE(args []string) []byte {
	if len(args) != 3 {
		return Encode(errors.New("ERR wrong number of arguments for 'zrange' command"), false)
	}

	key := args[0]
	start, err1 := strconv.Atoi(args[1])
	stop, err2 := strconv.Atoi(args[2])
	if err1 != nil || err2 != nil {
		return Encode(errors.New("ERR value is not an integer or out of range"), false)
	}

	obj := Get(key)
	if obj == nil {
		return Encode([]string{}, false) // empty array
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	zs := obj.Value.(*ZSet)
	members := zs.sl.Range(start, stop)
	return Encode(members, false)
}

// ZCARD key
func evalZCARD(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'zcard' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		return RESP_ZERO
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	zs := obj.Value.(*ZSet)
	return Encode(int64(len(zs.dict)), false)
}

// ZREM key member [member ...]
func evalZREM(args []string) []byte {
	if len(args) < 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'zrem' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		return RESP_ZERO
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	zs := obj.Value.(*ZSet)
	removed := 0

	for _, member := range args[1:] {
		if score, exists := zs.dict[member]; exists {
			zs.sl.Delete(score, member)
			delete(zs.dict, member)
			removed++
		}
	}

	if len(zs.dict) == 0 {
		Del(key)
	}

	return Encode(int64(removed), false)
}
