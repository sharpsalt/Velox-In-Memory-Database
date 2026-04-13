package config

var Host = "0.0.0.0"
var Port = 7379
var KeysLimit int=5

var EvictionStrategy string="simle-first"
var AOFile string="./dice-master.aof"