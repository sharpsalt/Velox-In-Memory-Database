/*
Set Commands — SADD, SREM, SMEMBERS, SISMEMBER, SCARD, SINTER, SUNION

Sets use two internal encodings:
  1. IntSet (OBJ_ENCODING_INTSET) — sorted int64 array for small integer-only sets
  2. Hashtable (OBJ_ENCODING_HT) — map[string]struct{} for large or mixed sets

Promotion from intset → hashtable happens when:
  - A non-integer member is added
  - The set exceeds SetMaxIntsetEntries (default 512)
*/
package core

import (
	"errors"
	"strconv"

	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

// setGetOrCreate retrieves the set object for key, or creates a new one.
// Tries intset first if the first member is an integer.
func setGetOrCreate(key string, firstMember string) (*Obj, []byte) {
	obj := Get(key)
	if obj == nil {
		// If first member is integer, start with intset; otherwise hashtable
		if _, err := strconv.ParseInt(firstMember, 10, 64); err == nil {
			obj = NewObj(NewIntSet(), -1, OBJ_TYPE_SET, OBJ_ENCODING_INTSET)
		} else {
			ht := make(map[string]struct{})
			obj = NewObj(ht, -1, OBJ_TYPE_SET, OBJ_ENCODING_HT)
		}
		Put(key, obj)
		return obj, nil
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_SET); err != nil {
		return nil, Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}
	return obj, nil
}

// setPromoteToHashtable converts an intset to hashtable encoding
func setPromoteToHashtable(obj *Obj) {
	is := obj.Value.(*IntSet)
	ht := make(map[string]struct{})
	for _, v := range is.Members() {
		ht[strconv.FormatInt(v, 10)] = struct{}{}
	}
	obj.Value = ht
	obj.TypeEncoding = OBJ_TYPE_SET | OBJ_ENCODING_HT
}

// setAdd adds a member to the set object, handling encoding promotion.
// Returns true if the member was new (not already in the set).
func setAdd(obj *Obj, member string) bool {
	enc := getEncoding(obj.TypeEncoding)

	if enc == OBJ_ENCODING_INTSET {
		is := obj.Value.(*IntSet)
		// Try to parse as integer
		val, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			// Non-integer → promote to hashtable
			setPromoteToHashtable(obj)
			ht := obj.Value.(map[string]struct{})
			if _, exists := ht[member]; exists {
				return false
			}
			ht[member] = struct{}{}
			return true
		}
		added := is.Add(val)
		// Check size threshold
		if is.Len() > config.SetMaxIntsetEntries {
			setPromoteToHashtable(obj)
		}
		return added
	}

	// Hashtable encoding
	ht := obj.Value.(map[string]struct{})
	if _, exists := ht[member]; exists {
		return false
	}
	ht[member] = struct{}{}
	return true
}

// SADD key member [member ...]
// Adds members to the set. Returns the number of members that were added (not duplicates).
func evalSADD(args []string) []byte {
	if len(args) < 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'sadd' command"), false)
	}

	key := args[0]
	obj, errResp := setGetOrCreate(key, args[1])
	if errResp != nil {
		return errResp
	}

	added := 0
	for _, member := range args[1:] {
		if setAdd(obj, member) {
			added++
		}
	}

	return Encode(int64(added), false)
}

// SREM key member [member ...]
// Removes members from the set. Returns the number of members removed.
func evalSREM(args []string) []byte {
	if len(args) < 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'srem' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		return RESP_ZERO
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_SET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	removed := 0
	enc := getEncoding(obj.TypeEncoding)

	for _, member := range args[1:] {
		switch enc {
		case OBJ_ENCODING_INTSET:
			is := obj.Value.(*IntSet)
			val, err := strconv.ParseInt(member, 10, 64)
			if err == nil && is.Remove(val) {
				removed++
			}
		case OBJ_ENCODING_HT:
			ht := obj.Value.(map[string]struct{})
			if _, exists := ht[member]; exists {
				delete(ht, member)
				removed++
			}
		}
	}

	// Delete key if set is now empty
	empty := false
	switch enc {
	case OBJ_ENCODING_INTSET:
		empty = obj.Value.(*IntSet).Len() == 0
	case OBJ_ENCODING_HT:
		empty = len(obj.Value.(map[string]struct{})) == 0
	}
	if empty {
		Del(key)
	}

	return Encode(int64(removed), false)
}

// SMEMBERS key
// Returns all members of the set as an array.
func evalSMEMBERS(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'smembers' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		return Encode([]string{}, false)
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_SET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	var members []string
	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_INTSET:
		is := obj.Value.(*IntSet)
		for _, v := range is.Members() {
			members = append(members, strconv.FormatInt(v, 10))
		}
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]struct{})
		members = make([]string, 0, len(ht))
		for k := range ht {
			members = append(members, k)
		}
	}
	if members == nil {
		members = []string{}
	}

	return Encode(members, false)
}

// SISMEMBER key member
// Returns 1 if the member is in the set, 0 otherwise.
func evalSISMEMBER(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'sismember' command"), false)
	}

	key, member := args[0], args[1]
	obj := Get(key)
	if obj == nil {
		return RESP_ZERO
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_SET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	exists := false
	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_INTSET:
		is := obj.Value.(*IntSet)
		val, err := strconv.ParseInt(member, 10, 64)
		if err == nil {
			exists = is.Contains(val)
		}
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]struct{})
		_, exists = ht[member]
	}

	if exists {
		return RESP_ONE
	}
	return RESP_ZERO
}

// SCARD key
// Returns the number of members in the set.
func evalSCARD(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'scard' command"), false)
	}

	key := args[0]
	obj := Get(key)
	if obj == nil {
		return RESP_ZERO
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_SET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_INTSET:
		return Encode(int64(obj.Value.(*IntSet).Len()), false)
	case OBJ_ENCODING_HT:
		return Encode(int64(len(obj.Value.(map[string]struct{}))), false)
	}
	return RESP_ZERO
}

// getSetMembers is a helper that extracts all members as map[string]struct{}
func getSetMembers(obj *Obj) map[string]struct{} {
	result := make(map[string]struct{})
	switch getEncoding(obj.TypeEncoding) {
	case OBJ_ENCODING_INTSET:
		is := obj.Value.(*IntSet)
		for _, v := range is.Members() {
			result[strconv.FormatInt(v, 10)] = struct{}{}
		}
	case OBJ_ENCODING_HT:
		ht := obj.Value.(map[string]struct{})
		for k := range ht {
			result[k] = struct{}{}
		}
	}
	return result
}

// SINTER key [key ...]
// Returns the intersection of all given sets.
func evalSINTER(args []string) []byte {
	if len(args) < 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'sinter' command"), false)
	}

	// Get first set
	obj := Get(args[0])
	if obj == nil {
		return Encode([]string{}, false)
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_SET); err != nil {
		return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	result := getSetMembers(obj)

	// Intersect with remaining sets
	for _, key := range args[1:] {
		obj2 := Get(key)
		if obj2 == nil {
			return Encode([]string{}, false)
		}
		if err := assertType(obj2.TypeEncoding, OBJ_TYPE_SET); err != nil {
			return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
		}
		other := getSetMembers(obj2)
		for k := range result {
			if _, exists := other[k]; !exists {
				delete(result, k)
			}
		}
	}

	members := make([]string, 0, len(result))
	for k := range result {
		members = append(members, k)
	}
	return Encode(members, false)
}

// SUNION key [key ...]
// Returns the union of all given sets.
func evalSUNION(args []string) []byte {
	if len(args) < 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'sunion' command"), false)
	}

	result := make(map[string]struct{})
	for _, key := range args {
		obj := Get(key)
		if obj == nil {
			continue
		}
		if err := assertType(obj.TypeEncoding, OBJ_TYPE_SET); err != nil {
			return Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
		}
		for k := range getSetMembers(obj) {
			result[k] = struct{}{}
		}
	}

	members := make([]string, 0, len(result))
	for k := range result {
		members = append(members, k)
	}
	return Encode(members, false)
}
