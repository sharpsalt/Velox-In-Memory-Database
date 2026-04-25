/*
Hash Commands — HSET, HGET, HGETALL, HDEL, HLEN, HEXISTS, HKEYS, HVALS, HINCRBY, HSETNX

Hashes use two internal encodings:
  1. ZipList (OBJ_ENCODING_ZIPLIST) — for small hashes (default)
     Compact, sequential storage. Field-value pairs stored as flat array.
  2. Hashtable (OBJ_ENCODING_HT) — for large hashes
     Go map[string]string. O(1) lookups.

Promotion from ziplist → hashtable happens when:
  - Number of fields exceeds HashMaxZiplistEntries (default 128)
  - Any field or value exceeds HashMaxZiplistValue bytes (default 64)

This matches exactly how Redis handles hash encoding internally.
*/
package core

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

// hashGetOrCreate retrieves the hash object for key, or creates a new one with ziplist encoding.
// Returns nil if the key exists but is not a hash type.
func hashGetOrCreate(key string) (*Obj, []byte) {
	obj := Get(key)
	if obj == nil {
		// Create new hash with ziplist encoding
		obj = NewObj(NewZipList(), -1, OBJ_TYPE_HASH, OBJ_ENCODING_ZIPLIST)
		Put(key, obj)
		return obj, nil
	}
	// Type check — must be a hash
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_HASH); err != nil {
		return nil, Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}
	return obj, nil
}

// hashCheckPromote checks if a ziplist-encoded hash should be promoted to hashtable.
// Call this after adding entries to a ziplist hash.
func hashCheckPromote(obj *Obj) {
	if getEncoding(obj.TypeEncoding) != OBJ_ENCODING_ZIPLIST {
		return
	}
	zl := obj.Value.(*ZipList)

	// Check entry count threshold
	if zl.HashLen() > config.HashMaxZiplistEntries {
		hashPromoteToHashtable(obj)
		return
	}

	// Check value size threshold
	entries := zl.HashGetAll()
	for _, entry := range entries {
		if len(entry) > config.HashMaxZiplistValue {
			hashPromoteToHashtable(obj)
			return
		}
	}
}

// hashPromoteToHashtable converts a ziplist hash to hashtable encoding
func hashPromoteToHashtable(obj *Obj) {
	zl := obj.Value.(*ZipList)
	ht := zl.HashEntries()
	obj.Value = ht
	obj.TypeEncoding = OBJ_TYPE_HASH | OBJ_ENCODING_HT
}

// HSET key field value [field value ...]
// Sets one or more field-value pairs. Returns the number of NEW fields added.
func evalHSET(args []string) []byte {
	if len(args) < 3 || len(args)%2 == 0 {
		return Encode(errors.New("ERR wrong number of arguments for 'hset' command"), false)
	}

	key := args[0]
	obj, errResp := hashGetOrCreate(key)
	if errResp != nil {
		return errResp
	}

	added := 0
	enc := getEncoding(obj.TypeEncoding)

	for i := 1; i < len(args)-1; i += 2 {
		field, value := args[i], args[i+1]

		switch enc {
		case OBJ_ENCODING_ZIPLIST:
			zl := obj.Value.(*ZipList)
			if zl.HashSet(field, value) {
				added++
			}
		case OBJ_ENCODING_HT:
			ht := obj.Value.(map[string]string)
			if _, exists := ht[field]; !exists {
				added++
			}
			ht[field] = value
		}
	}

	// Check if we need to promote encoding
	hashCheckPromote(obj)

	return Encode(int64(added), false)
}

// HGET key field
// Returns the value of a hash field, or nil if field/key doesn't exist.
func evalHGET(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'hget' command"), false)
	}

	key, field := args[0], args[1]
	obj := Get(key)
	if obj == nil {
		return RESP_NIL
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_HASH); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_ZIPLIST:
		zl := obj.Value.(*ZipList)
		val, ok := zl.HashGet(field)
		if !ok {
			return RESP_NIL
		}
		return Encode(val, false)
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]string)
		val, ok := ht[field]
		if !ok {
			return RESP_NIL
		}
		return Encode(val, false)
	}
	return RESP_NIL
}

// HGETALL key
// Returns all field-value pairs as an array: [field1, value1, field2, value2, ...]
func evalHGETALL(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'hgetall' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		// Return empty array for non-existent key
		return Encode([]string{}, false)
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_HASH); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	var result []string
	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_ZIPLIST:
		zl := obj.Value.(*ZipList)
		result = zl.HashGetAll()
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]string)
		result = make([]string, 0, len(ht)*2)
		for k, v := range ht {
			result = append(result, k, v)
		}
	}

	return Encode(result, false)
}

// HDEL key field [field ...]
// Removes one or more fields from a hash. Returns the number of fields removed.
func evalHDEL(args []string) []byte {
	if len(args) < 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'hdel' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		return RESP_ZERO
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_HASH); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	deleted := 0
	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_ZIPLIST:
		zl := obj.Value.(*ZipList)
		for _, field := range args[1:] {
			if zl.HashDel(field) {
				deleted++
			}
		}
		// If hash is now empty, delete the key
		if zl.HashLen() == 0 {
			Del(key)
		}
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]string)
		for _, field := range args[1:] {
			if _, exists := ht[field]; exists {
				delete(ht, field)
				deleted++
			}
		}
		if len(ht) == 0 {
			Del(key)
		}
	}

	return Encode(int64(deleted), false)
}

// HLEN key
// Returns the number of fields in the hash.
func evalHLEN(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'hlen' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		return RESP_ZERO
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_HASH); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_ZIPLIST:
		zl := obj.Value.(*ZipList)
		return Encode(int64(zl.HashLen()), false)
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]string)
		return Encode(int64(len(ht)), false)
	}
	return RESP_ZERO
}

// HEXISTS key field
// Returns 1 if the field exists, 0 if not.
func evalHEXISTS(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'hexists' command"), false)
	}

	key, field := args[0], args[1]
	obj := Get(key)
	if obj == nil {
		return RESP_ZERO
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_HASH); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	exists := false
	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_ZIPLIST:
		zl := obj.Value.(*ZipList)
		exists = zl.HashExists(field)
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]string)
		_, exists = ht[field]
	}

	if exists {
		return RESP_ONE
	}
	return RESP_ZERO
}

// HKEYS key
// Returns all field names in the hash.
func evalHKEYS(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'hkeys' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		return Encode([]string{}, false)
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_HASH); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	var keys []string
	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_ZIPLIST:
		zl := obj.Value.(*ZipList)
		keys = zl.HashKeys()
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]string)
		keys = make([]string, 0, len(ht))
		for k := range ht {
			keys = append(keys, k)
		}
	}

	return Encode(keys, false)
}

// HVALS key
// Returns all values in the hash.
func evalHVALS(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'hvals' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		return Encode([]string{}, false)
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_HASH); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	var vals []string
	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_ZIPLIST:
		zl := obj.Value.(*ZipList)
		vals = zl.HashValues()
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]string)
		vals = make([]string, 0, len(ht))
		for _, v := range ht {
			vals = append(vals, v)
		}
	}

	return Encode(vals, false)
}

// HINCRBY key field increment
// Increments the integer value of a hash field by the given increment.
// If the field doesn't exist, it's created with value 0 before incrementing.
func evalHINCRBY(args []string) []byte {
	if len(args) != 3 {
		return Encode(errors.New("ERR wrong number of arguments for 'hincrby' command"), false)
	}

	key, field := args[0], args[1]
	incr, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return Encode(errors.New("ERR value is not an integer or out of range"), false)
	}

	obj, errResp := hashGetOrCreate(key)
	if errResp != nil {
		return errResp
	}

	var currentVal int64

	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_ZIPLIST:
		zl := obj.Value.(*ZipList)
		valStr, exists := zl.HashGet(field)
		if exists {
			currentVal, err = strconv.ParseInt(valStr, 10, 64)
			if err != nil {
				return Encode(errors.New("ERR hash value is not an integer"), false)
			}
		}
		currentVal += incr
		zl.HashSet(field, fmt.Sprintf("%d", currentVal))
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]string)
		valStr, exists := ht[field]
		if exists {
			currentVal, err = strconv.ParseInt(valStr, 10, 64)
			if err != nil {
				return Encode(errors.New("ERR hash value is not an integer"), false)
			}
		}
		currentVal += incr
		ht[field] = fmt.Sprintf("%d", currentVal)
	}

	hashCheckPromote(obj)
	return Encode(currentVal, false)
}

// HSETNX key field value
// Set field only if it does NOT already exist. Returns 1 if set, 0 if not.
func evalHSETNX(args []string) []byte {
	if len(args) != 3 {
		return Encode(errors.New("ERR wrong number of arguments for 'hsetnx' command"), false)
	}

	key, field, value := args[0], args[1], args[2]
	obj, errResp := hashGetOrCreate(key)
	if errResp != nil {
		return errResp
	}

	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_ZIPLIST:
		zl := obj.Value.(*ZipList)
		if zl.HashExists(field) {
			return RESP_ZERO
		}
		zl.HashSet(field, value)
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]string)
		if _, exists := ht[field]; exists {
			return RESP_ZERO
		}
		ht[field] = value
	}

	hashCheckPromote(obj)
	return RESP_ONE
}
