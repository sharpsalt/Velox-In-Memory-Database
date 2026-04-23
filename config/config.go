package config

var Host = "0.0.0.0"
var Port = 7379
var KeysLimit int = 5

//will evict EvictionRatio of keys whenever eviction runs
//it would dictate whenever my eviction is triggered, how many keys i will be evicting.
var EvictionRatio float64 = 0.40 //Generally aise real scenario me kabhi itna to nahi hi krte hai but still

var EvictionStrategy string = "simle-first"
var AOFile string = "./velox.aof"