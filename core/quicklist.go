/*
QuickList — A doubly-linked list of ZipList nodes.

This is the ACTUAL data structure Redis uses for all lists (since Redis 3.2).
It combines the best of both worlds:
  - Linked list: O(1) push/pop at both ends
  - ZipList: cache-friendly, compact storage for small runs of elements

Each QuickList node contains a ZipList. When a node's ziplist exceeds
ListMaxZiplistSize entries, new elements go into a new node rather than
growing the existing ziplist indefinitely.

Why not just use a Go slice?
  - Slice LPUSH is O(n) because it shifts all elements right
  - QuickList LPUSH is O(1) — just prepend to the head node's ziplist
  - For large lists (millions of elements), this difference is massive
*/
package core

import (
	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

// QuickListNode is a single node in the doubly-linked list.
// Each node contains a ZipList of elements.
type QuickListNode struct {
	zl   *ZipList
	prev *QuickListNode
	next *QuickListNode
}

// QuickList is a doubly-linked list of ZipList nodes.
type QuickList struct {
	head  *QuickListNode
	tail  *QuickListNode
	count int // total elements across ALL nodes
	len   int // number of QuickListNode nodes
}

// NewQuickList creates an empty QuickList
func NewQuickList() *QuickList {
	return &QuickList{}
}

// Len returns the total number of elements across all nodes
func (ql *QuickList) Len() int {
	return ql.count
}

// ensureHead makes sure there's at least one node in the list
func (ql *QuickList) ensureHead() {
	if ql.head == nil {
		node := &QuickListNode{zl: NewZipList()}
		ql.head = node
		ql.tail = node
		ql.len = 1
	}
}

// LPush prepends elements to the left (head) of the list.
// Elements are added in order, so LPUSH key a b c results in [c, b, a, ...].
func (ql *QuickList) LPush(values ...string) {
	ql.ensureHead()
	for _, val := range values {
		// If head node's ziplist is full, create a new head node
		if ql.head.zl.Len() >= config.ListMaxZiplistSize {
			newNode := &QuickListNode{zl: NewZipList()}
			newNode.next = ql.head
			ql.head.prev = newNode
			ql.head = newNode
			ql.len++
		}
		ql.head.zl.LPush(val)
		ql.count++
	}
}

// RPush appends elements to the right (tail) of the list.
func (ql *QuickList) RPush(values ...string) {
	ql.ensureHead()
	for _, val := range values {
		// If tail node's ziplist is full, create a new tail node
		if ql.tail.zl.Len() >= config.ListMaxZiplistSize {
			newNode := &QuickListNode{zl: NewZipList()}
			newNode.prev = ql.tail
			ql.tail.next = newNode
			ql.tail = newNode
			ql.len++
		}
		ql.tail.zl.RPush(val)
		ql.count++
	}
}

// LPop removes and returns the leftmost (head) element.
// Returns ("", false) if the list is empty.
func (ql *QuickList) LPop() (string, bool) {
	if ql.head == nil {
		return "", false
	}

	val, ok := ql.head.zl.LPop()
	if !ok {
		return "", false
	}
	ql.count--

	// If head node is now empty, remove it
	if ql.head.zl.Len() == 0 {
		ql.removeNode(ql.head)
	}

	return val, true
}

// RPop removes and returns the rightmost (tail) element.
// Returns ("", false) if the list is empty.
func (ql *QuickList) RPop() (string, bool) {
	if ql.tail == nil {
		return "", false
	}

	val, ok := ql.tail.zl.RPop()
	if !ok {
		return "", false
	}
	ql.count--

	// If tail node is now empty, remove it
	if ql.tail.zl.Len() == 0 {
		ql.removeNode(ql.tail)
	}

	return val, true
}

// removeNode removes a node from the linked list.
func (ql *QuickList) removeNode(node *QuickListNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		ql.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		ql.tail = node.prev
	}
	ql.len--
	if ql.len == 0 {
		ql.head = nil
		ql.tail = nil
	}
}

// Index returns the element at the given index.
// Supports negative indices (-1 = last element).
// O(n) in worst case but walks node-by-node skipping ziplist sizes.
func (ql *QuickList) Index(idx int) (string, bool) {
	if ql.count == 0 {
		return "", false
	}

	// Handle negative indices
	if idx < 0 {
		idx = ql.count + idx
	}
	if idx < 0 || idx >= ql.count {
		return "", false
	}

	// Walk from head, skipping whole nodes when possible
	node := ql.head
	for node != nil {
		nodeLen := node.zl.Len()
		if idx < nodeLen {
			return node.zl.Index(idx)
		}
		idx -= nodeLen
		node = node.next
	}
	return "", false
}

// Range returns elements from start to stop (inclusive).
// Supports negative indices like Redis LRANGE.
func (ql *QuickList) Range(start, stop int) []string {
	if ql.count == 0 {
		return []string{}
	}

	// Convert negative indices
	if start < 0 {
		start = ql.count + start
	}
	if stop < 0 {
		stop = ql.count + stop
	}
	// Clamp
	if start < 0 {
		start = 0
	}
	if stop >= ql.count {
		stop = ql.count - 1
	}
	if start > stop {
		return []string{}
	}

	result := make([]string, 0, stop-start+1)
	pos := 0 // current global position

	node := ql.head
	for node != nil && pos <= stop {
		nodeLen := node.zl.Len()
		nodeEnd := pos + nodeLen - 1

		if nodeEnd >= start {
			// This node contains some of our range
			localStart := 0
			if start > pos {
				localStart = start - pos
			}
			localStop := nodeLen - 1
			if stop < nodeEnd {
				localStop = stop - pos
			}

			result = append(result, node.zl.Range(localStart, localStop)...)
		}

		pos += nodeLen
		node = node.next
	}

	return result
}

// AllElements returns all elements in order (for serialization/AOF dump)
func (ql *QuickList) AllElements() []string {
	result := make([]string, 0, ql.count)
	node := ql.head
	for node != nil {
		result = append(result, node.zl.Entries()...)
		node = node.next
	}
	return result
}
