package core

import (
	"math/rand"
)

/*
SkipList — The core data structure for Redis ZSet (Sorted Set)

A Skip List is a probabilistic data structure that provides O(log N) expected time
for search, insertion, and deletion. It's built on top of multiple layers of linked
lists, where each higher layer acts as an "express lane" for the lists below it.

Why use Skip List instead of a Balanced Tree (e.g. Red-Black Tree)?
1. Simpler to implement and maintain.
2. Faster for concurrent/ordered range queries (just walk the bottom linked list).
3. Less memory overhead per node on average.
4. Redis uses Skip Lists for Sorted Sets.

Our implementation matches the Redis architecture:
- Nodes contain a string member and a float64 score.
- Nodes are sorted by score ascending. If scores are equal, sorted lexicographically by member.
- Back pointers allow reverse traversal (for ZREVRANGE).
- Span fields track the distance between nodes at each level, allowing O(log N) rank queries (ZRANK).
*/

const (
	SkipListMaxLevel = 32
	SkipListP        = 0.25 // Probability for generating levels
)

type SkipListLevel struct {
	forward *SkipListNode
	span    int // The number of nodes skipped to reach the forward node
}

type SkipListNode struct {
	member   string
	score    float64
	backward *SkipListNode
	level    []SkipListLevel // Array of levels
}

type SkipList struct {
	head   *SkipListNode
	tail   *SkipListNode
	length int
	level  int
}

// randomLevel generates a random level for a new node.
// It returns a level between 1 and SkipListMaxLevel.
// The probability of each level is (SkipListP) ^ (level - 1).
func randomLevel() int {
	level := 1
	for float64(rand.Int()&0xFFFF) < float64(0xFFFF)*SkipListP {
		level++
	}
	if level > SkipListMaxLevel {
		level = SkipListMaxLevel
	}
	return level
}

// CreateSkipList initializes and returns an empty skip list.
func CreateSkipList() *SkipList {
	// Head node has no member/score, just levels
	head := &SkipListNode{
		member: "",
		score:  0,
		level:  make([]SkipListLevel, SkipListMaxLevel),
	}
	return &SkipList{
		head:   head,
		tail:   nil,
		length: 0,
		level:  1,
	}
}

// Insert adds a new member with the given score to the skip list.
func (sl *SkipList) Insert(score float64, member string) *SkipListNode {
	update := make([]*SkipListNode, SkipListMaxLevel)
	rank := make([]int, SkipListMaxLevel)

	x := sl.head
	// Find the insertion point
	for i := sl.level - 1; i >= 0; i-- {
		if i == sl.level-1 {
			rank[i] = 0
		} else {
			rank[i] = rank[i+1]
		}

		// Traverse forward while score is less, or score is equal and member is less
		for x.level[i].forward != nil &&
			(x.level[i].forward.score < score ||
				(x.level[i].forward.score == score && x.level[i].forward.member < member)) {
			rank[i] += x.level[i].span
			x = x.level[i].forward
		}
		update[i] = x
	}

	// We don't check for exact member existence here; ZSet implementation
	// should use a hashtable (map) alongside the skiplist to enforce uniqueness.

	level := randomLevel()
	if level > sl.level {
		for i := sl.level; i < level; i++ {
			rank[i] = 0
			update[i] = sl.head
			update[i].level[i].span = sl.length
		}
		sl.level = level
	}

	x = &SkipListNode{
		member: member,
		score:  score,
		level:  make([]SkipListLevel, level),
	}

	// Update pointers and spans
	for i := 0; i < level; i++ {
		x.level[i].forward = update[i].level[i].forward
		update[i].level[i].forward = x

		x.level[i].span = update[i].level[i].span - (rank[0] - rank[i])
		update[i].level[i].span = (rank[0] - rank[i]) + 1
	}

	// Increment span for untouched levels
	for i := level; i < sl.level; i++ {
		update[i].level[i].span++
	}

	// Set backward pointer
	if update[0] == sl.head {
		x.backward = nil
	} else {
		x.backward = update[0]
	}

	if x.level[0].forward != nil {
		x.level[0].forward.backward = x
	} else {
		sl.tail = x
	}

	sl.length++
	return x
}

// deleteNode removes a node from the skip list given the update array.
func (sl *SkipList) deleteNode(x *SkipListNode, update []*SkipListNode) {
	for i := 0; i < sl.level; i++ {
		if update[i].level[i].forward == x {
			update[i].level[i].span += x.level[i].span - 1
			update[i].level[i].forward = x.level[i].forward
		} else {
			update[i].level[i].span--
		}
	}

	if x.level[0].forward != nil {
		x.level[0].forward.backward = x.backward
	} else {
		sl.tail = x.backward
	}

	for sl.level > 1 && sl.head.level[sl.level-1].forward == nil {
		sl.level--
	}
	sl.length--
}

// Delete removes a member with the given score.
func (sl *SkipList) Delete(score float64, member string) bool {
	update := make([]*SkipListNode, SkipListMaxLevel)
	x := sl.head

	for i := sl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil &&
			(x.level[i].forward.score < score ||
				(x.level[i].forward.score == score && x.level[i].forward.member < member)) {
			x = x.level[i].forward
		}
		update[i] = x
	}

	x = x.level[0].forward
	if x != nil && x.score == score && x.member == member {
		sl.deleteNode(x, update)
		return true
	}
	return false
}

// GetRank returns the 1-based rank of the given score/member. Returns 0 if not found.
func (sl *SkipList) GetRank(score float64, member string) int {
	rank := 0
	x := sl.head

	for i := sl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil &&
			(x.level[i].forward.score < score ||
				(x.level[i].forward.score == score && x.level[i].forward.member <= member)) {
			rank += x.level[i].span
			x = x.level[i].forward
		}
		if x.member == member {
			return rank
		}
	}
	return 0
}

// GetNodeByRank returns the node at the 1-based rank.
func (sl *SkipList) GetNodeByRank(rank int) *SkipListNode {
	if rank == 0 || rank > sl.length {
		return nil
	}

	traversed := 0
	x := sl.head

	for i := sl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && traversed+x.level[i].span <= rank {
			traversed += x.level[i].span
			x = x.level[i].forward
		}
		if traversed == rank {
			return x
		}
	}
	return nil
}

// Range returns the members within the 0-based index range [start, end].
// Supports negative indices (-1 = last element).
func (sl *SkipList) Range(start, end int) []string {
	// Normalize negative indices
	if start < 0 {
		start = sl.length + start
	}
	if end < 0 {
		end = sl.length + end
	}
	if start < 0 {
		start = 0
	}
	if start > end || start >= sl.length {
		return []string{}
	}
	if end >= sl.length {
		end = sl.length - 1
	}

	// Ranks are 1-based, indices are 0-based
	node := sl.GetNodeByRank(start + 1)
	if node == nil {
		return []string{}
	}

	count := end - start + 1
	result := make([]string, 0, count)

	for i := 0; i < count && node != nil; i++ {
		result = append(result, node.member)
		node = node.level[0].forward
	}

	return result
}
