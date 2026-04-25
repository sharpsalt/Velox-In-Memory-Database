/*
IntSet — A sorted array of unique integers for small integer-only sets.

In Redis, IntSet is a memory-efficient encoding for sets that contain ONLY integers.
It stores integers in a sorted array and uses binary search for O(log n) lookups.

Redis supports 3 encoding widths (int16, int32, int64) and upgrades dynamically.
Our Go implementation uses int64 throughout for simplicity — Go slices already handle
memory efficiently and the additional complexity of encoding upgrades isn't worth it
for our use case.

When a non-integer member is added to a set, or when the set exceeds
SetMaxIntsetEntries, the set is promoted from IntSet to a hashtable encoding
(map[string]struct{}).
*/
package core

import "sort"

// IntSet stores unique integers in a sorted slice.
// Binary search is used for all lookups → O(log n).
type IntSet struct {
	contents []int64 // always sorted in ascending order
}

// NewIntSet creates a new empty IntSet
func NewIntSet() *IntSet {
	return &IntSet{
		contents: make([]int64, 0),
	}
}

// Len returns the number of integers in the set
func (is *IntSet) Len() int {
	return len(is.contents)
}

// search returns the index where val is or would be inserted (binary search)
func (is *IntSet) search(val int64) (int, bool) {
	idx := sort.Search(len(is.contents), func(i int) bool {
		return is.contents[i] >= val
	})
	if idx < len(is.contents) && is.contents[idx] == val {
		return idx, true
	}
	return idx, false
}

// Add inserts a value into the sorted set.
// Returns true if the value was added (not a duplicate).
func (is *IntSet) Add(val int64) bool {
	idx, found := is.search(val)
	if found {
		return false // already exists
	}
	// Insert at idx to maintain sorted order
	is.contents = append(is.contents, 0)
	copy(is.contents[idx+1:], is.contents[idx:])
	is.contents[idx] = val
	return true
}

// Remove deletes a value from the set.
// Returns true if the value was found and removed.
func (is *IntSet) Remove(val int64) bool {
	idx, found := is.search(val)
	if !found {
		return false
	}
	is.contents = append(is.contents[:idx], is.contents[idx+1:]...)
	return true
}

// Contains checks if a value exists in the set. O(log n).
func (is *IntSet) Contains(val int64) bool {
	_, found := is.search(val)
	return found
}

// Members returns all integers as a copy of the sorted slice.
func (is *IntSet) Members() []int64 {
	result := make([]int64, len(is.contents))
	copy(result, is.contents)
	return result
}
