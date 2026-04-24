package config

// Duplicate variables - commenting out to avoid redeclaration
// See config.go for the actual declarations
var Host string="0.0.0.0"
var Port int=7379

//var KeysLimit int=100 //like ye hum threshold set krdiye hai like our database still support atmax this many keys 

//will evict EvictionRation of keys whenever evictionruns
//it would dictate whenever my eviction is triggered, how many keys i will be evicting.
var EvictionRatio float64=0.40 //Generally aise real scenario me kabhi itna to nahi hi krte hai but still

var EvictionStrategy string="allkeys-random"
var AOFFile string="./velox.aof"

