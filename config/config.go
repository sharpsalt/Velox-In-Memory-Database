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