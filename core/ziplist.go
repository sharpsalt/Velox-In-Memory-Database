/*
ZipList — A compact, sequential storage for small collections.

In Redis, a ziplist is a packed byte array where entries are stored contiguously
in memory with variable-length encoding. This gives excellent cache locality and
minimal memory overhead for small collections (< ~128 entries).

Our Go implementation uses []string for clarity while preserving the same semantics:
- Sequential scan for lookups (O(n) but fast for small n due to cache locality)
- No per-entry pointer overhead like a map or linked list
- Used as backing store for small hashes (field-value pairs stored flat)
- Used as nodes inside QuickList for list data structures

When a ziplist-backed hash grows beyond HashMaxZiplistEntries or any value
exceeds HashMaxZiplistValue bytes, it gets promoted to a hashtable encoding.
*/
package core

// ZipList stores entries as a flat string slice.
// For hashes: [field1, value1, field2, value2, ...]
// For lists/quicklist nodes: [elem1, elem2, elem3, ...]
type ZipList struct {
	entries []string
}

// NewZipList creates a new empty ZipList
func NewZipList() *ZipList {
	return &ZipList{
		entries: make([]string, 0),
	}
}

// Len returns the number of raw entries in the ziplist
func (zl *ZipList) Len() int {
	return len(zl.entries)
}

// --- Hash operations (field-value pairs stored flat) ---

// HashLen returns the number of field-value pairs (entries/2)
func (zl *ZipList) HashLen() int {
	return len(zl.entries) / 2
}

// HashGet looks up a field and returns its value.
// Returns (value, true) if found, ("", false) otherwise.
// O(n) scan — fast for small hashes.
func (zl *ZipList) HashGet(field string) (string, bool) {
	for i := 0; i < len(zl.entries)-1; i += 2 {
		if zl.entries[i] == field {
			return zl.entries[i+1], true
		}
	}
	return "", false
}

// HashSet sets a field-value pair. If the field already exists, it updates the value.
// Returns true if a new field was added, false if an existing field was updated.
func (zl *ZipList) HashSet(field, value string) bool {
	for i := 0; i < len(zl.entries)-1; i += 2 {
		if zl.entries[i] == field {
			zl.entries[i+1] = value
			return false // updated existing
		}
	}
	// New field — append at the end
	zl.entries = append(zl.entries, field, value)
	return true // added new
}

// HashDel removes a field-value pair by field name.
// Returns true if the field was found and deleted.
func (zl *ZipList) HashDel(field string) bool {
	for i := 0; i < len(zl.entries)-1; i += 2 {
		if zl.entries[i] == field {
			// Remove both field and value (2 entries)
			zl.entries = append(zl.entries[:i], zl.entries[i+2:]...)
			return true
		}
	}
	return false
}

// HashExists checks if a field exists in the ziplist hash.
func (zl *ZipList) HashExists(field string) bool {
	for i := 0; i < len(zl.entries)-1; i += 2 {
		if zl.entries[i] == field {
			return true
		}
	}
	return false
}

// HashEntries returns all field-value pairs as a map.
// Used during promotion from ziplist to hashtable encoding.
func (zl *ZipList) HashEntries() map[string]string {
	m := make(map[string]string, len(zl.entries)/2)
	for i := 0; i < len(zl.entries)-1; i += 2 {
		m[zl.entries[i]] = zl.entries[i+1]
	}
	return m
}

// HashKeys returns all field names.
func (zl *ZipList) HashKeys() []string {
	keys := make([]string, 0, len(zl.entries)/2)
	for i := 0; i < len(zl.entries)-1; i += 2 {
		keys = append(keys, zl.entries[i])
	}
	return keys
}

// HashValues returns all values.
func (zl *ZipList) HashValues() []string {
	vals := make([]string, 0, len(zl.entries)/2)
	for i := 0; i < len(zl.entries)-1; i += 2 {
		vals = append(vals, zl.entries[i+1])
	}
	return vals
}

// HashGetAll returns all entries as a flat array [field1, value1, field2, value2, ...]
// This is the format needed for HGETALL RESP response.
func (zl *ZipList) HashGetAll() []string {
	result := make([]string, len(zl.entries))
	copy(result, zl.entries)
	return result
}

// --- List operations (sequential elements) ---

// LPush prepends elements to the front of the ziplist (leftmost = index 0)
func (zl *ZipList) LPush(values ...string) {
	// Prepend: new entries go to the front
	zl.entries = append(values, zl.entries...)
}

// RPush appends elements to the end of the ziplist
func (zl *ZipList) RPush(values ...string) {
	zl.entries = append(zl.entries, values...)
}

// LPop removes and returns the first element.
// Returns ("", false) if empty.
func (zl *ZipList) LPop() (string, bool) {
	if len(zl.entries) == 0 {
		return "", false
	}
	val := zl.entries[0]
	zl.entries = zl.entries[1:]
	return val, true
}

// RPop removes and returns the last element.
// Returns ("", false) if empty.
func (zl *ZipList) RPop() (string, bool) {
	if len(zl.entries) == 0 {
		return "", false
	}
	val := zl.entries[len(zl.entries)-1]
	zl.entries = zl.entries[:len(zl.entries)-1]
	return val, true
}

// Index returns the element at the given index.
// Supports negative indices (-1 = last element).
// Returns ("", false) if index is out of range.
func (zl *ZipList) Index(idx int) (string, bool) {
	n := len(zl.entries)
	if idx < 0 {
		idx = n + idx
	}
	if idx < 0 || idx >= n {
		return "", false
	}
	return zl.entries[idx], true
}

// Range returns elements from start to stop (inclusive).
// Supports negative indices like Redis LRANGE.
func (zl *ZipList) Range(start, stop int) []string {
	n := len(zl.entries)
	if n == 0 {
		return []string{}
	}

	// Convert negative indices
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}

	// Clamp to valid range
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop {
		return []string{}
	}

	result := make([]string, stop-start+1)
	copy(result, zl.entries[start:stop+1])
	return result
}

// Entries returns a copy of all entries (for serialization)
func (zl *ZipList) Entries() []string {
	result := make([]string, len(zl.entries))
	copy(result, zl.entries)
	return result
}
