package config

var Host = "0.0.0.0"
var Port = 7379
var KeysLimit int = 100

//will evict EvictionRatio of keys whenever eviction runs
//it would dictate whenever my eviction is triggered, how many keys i will be evicting.
var EvictionRatio float64 = 0.40 //Generally aise real scenario me kabhi itna to nahi hi krte hai but still

// EvictionStrategy controls which eviction algorithm is used when KeysLimit is reached.
// Options:
//   "simple-first"   — evicts the first key found (fastest but worst accuracy)
//   "allkeys-random" — evicts random keys
//   "allkeys-lru"    — approximated LRU using sampling + eviction pool (recommended)
var EvictionStrategy string = "allkeys-lru"

// LRUSampleSize is the number of random keys sampled on each eviction round.
// Redis default is 5. Higher = more accurate LRU but slower eviction.
// With 10 samples, approximated LRU is very close to true LRU.
var LRUSampleSize int = 5

var AOFFile string = "./velox.aof"

// Hash encoding promotion thresholds
// When a hash exceeds these limits, it converts from ziplist to hashtable
var HashMaxZiplistEntries int = 128  // max number of field-value pairs in ziplist encoding
var HashMaxZiplistValue int = 64     // max byte length of any field or value in ziplist encoding

// List: max entries per ziplist node inside the quicklist
var ListMaxZiplistSize int = 128

// Set encoding promotion threshold
// When a set exceeds this many entries, or gets a non-integer member, it converts from intset to hashtable
var SetMaxIntsetEntries int = 512