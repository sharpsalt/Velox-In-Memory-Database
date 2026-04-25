package core

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)


var RESP_NIL []byte=[]byte("$-1\r\n")
var RESP_OK []byte=[]byte("+OK\r\n")
var RESP_ZERO []byte=[]byte(":0\r\n")
var RESP_ONE []byte=[]byte(":1\r\n")
var RESP_MINUS_1 []byte=[]byte(":-1\r\n")
var RESP_MINUS_2 []byte=[]byte(":-2\r\n")
var RESP_QUEUED []byte=[]byte("+QUEUED\r\n")

var txnCommands map[string]bool

func init(){
	txnCommands=map[string]bool{"EXEC":true,"DISCARD":true}
}

// func evalPING(args []string,c net.Conn)error{
func evalPING(args []string) []byte{
	//eariler we used to return an error , but now we return a slice of bytes(which is the actual response)
	var b []byte 

	if len(args) >= 2{
		//means if the redis cli passes us more than 1 arguments then this will invoke 
		return Encode(errors.New("ERR wrong number of arguments for 'ping' command"), false)
	}

	if len(args) == 0{
		//we will encode it into RESP
		//encode function is to take the raw type and convert it to another encoded resp format
		//Because server has to respond in resp format so the server will also do get the thing
		b = Encode("PONG", true)
	}else{
		b = Encode(args[0], false)
	}

	// _,err:=c.Write(b)
	return b
}


func evalINCR(args []string)[]byte{
	if len(args)!=1{
		return Encode(errors.New("ERR wrong number of arguments for 'incr' command"),false)
	}

	var key string=args[0] //first wala argument lenge hum isme 
	obj:=Get(key)
	//basically we will get an object and if that obejct doesn;t exits then we will create a new object
	if obj==nil{
		obj=NewObj("0",-1,OBJ_TYPE_STRING,OBJ_ENCODING_INT)
		Put(key,obj)
	}

	if err:=assertType(obj.TypeEncoding,OBJ_TYPE_STRING);err!=nil{
		return Encode(err,false)
	}

	if err:=assertEncoding(obj.TypeEncoding,OBJ_ENCODING_INT);err!=nil{
		return Encode(err,false)
	}
	//which means value is indeed an integer
	i, _ := strconv.ParseInt(obj.Value.(string), 10, 64)  // Fixed: proper type assertion
	i++
	obj.Value = strconv.FormatInt(i, 10)

	return Encode(i, false)
}

// DECR key — decrements the integer value by 1
func evalDECR(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'decr' command"), false)
	}
	key := args[0]
	obj := Get(key)
	if obj == nil {
		obj = NewObj("0", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT)
		Put(key, obj)
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_STRING); err != nil {
		return Encode(err, false)
	}
	if err := assertEncoding(obj.TypeEncoding, OBJ_ENCODING_INT); err != nil {
		return Encode(err, false)
	}
	i, _ := strconv.ParseInt(obj.Value.(string), 10, 64)
	i--
	obj.Value = strconv.FormatInt(i, 10)
	return Encode(i, false)
}

// INCRBY key increment — increments the integer value by the given amount
func evalINCRBY(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'incrby' command"), false)
	}
	key := args[0]
	incr, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return Encode(errors.New("ERR value is not an integer or out of range"), false)
	}
	obj := Get(key)
	if obj == nil {
		obj = NewObj("0", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT)
		Put(key, obj)
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_STRING); err != nil {
		return Encode(err, false)
	}
	if err := assertEncoding(obj.TypeEncoding, OBJ_ENCODING_INT); err != nil {
		return Encode(err, false)
	}
	i, _ := strconv.ParseInt(obj.Value.(string), 10, 64)
	i += incr
	obj.Value = strconv.FormatInt(i, 10)
	return Encode(i, false)
}

// DECRBY key decrement — decrements the integer value by the given amount
func evalDECRBY(args []string) []byte {
	if len(args) != 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'decrby' command"), false)
	}
	key := args[0]
	decr, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return Encode(errors.New("ERR value is not an integer or out of range"), false)
	}
	obj := Get(key)
	if obj == nil {
		obj = NewObj("0", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT)
		Put(key, obj)
	}
	if err := assertType(obj.TypeEncoding, OBJ_TYPE_STRING); err != nil {
		return Encode(err, false)
	}
	if err := assertEncoding(obj.TypeEncoding, OBJ_ENCODING_INT); err != nil {
		return Encode(err, false)
	}
	i, _ := strconv.ParseInt(obj.Value.(string), 10, 64)
	i -= decr
	obj.Value = strconv.FormatInt(i, 10)
	return Encode(i, false)
}

// MSET key1 value1 key2 value2 ... — sets multiple key-value pairs atomically
func evalMSET(args []string) []byte {
	if len(args) < 2 || len(args)%2 != 0 {
		return Encode(errors.New("ERR wrong number of arguments for 'mset' command"), false)
	}
	for i := 0; i < len(args); i += 2 {
		key, value := args[i], args[i+1]
		otype, oEnc := deduceTypeEncoding(value)
		Put(key, NewObj(value, -1, otype, oEnc))
	}
	return RESP_OK
}

// MGET key1 key2 ... — returns the values for all given keys
func evalMGET(args []string) []byte {
	if len(args) < 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'mget' command"), false)
	}
	results := make([]string, len(args))
	for i, key := range args {
		obj := Get(key)
		if obj == nil || hasExpired(obj) {
			results[i] = "(nil)"
		} else {
			results[i] = fmt.Sprintf("%v", obj.Value)
		}
	}
	return Encode(results, false)
}

// EXISTS key [key ...] — returns the count of how many of the given keys exist
func evalEXISTS(args []string) []byte {
	if len(args) < 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'exists' command"), false)
	}
	count := 0
	for _, key := range args {
		obj := Get(key)
		if obj != nil {
			count++
		}
	}
	return Encode(int64(count), false)
}

// KEYS pattern — returns all keys matching the given glob pattern
// Uses Go's path.Match for glob matching (supports *, ?, [...])
func evalKEYS(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'keys' command"), false)
	}
	pattern := args[0]
	var matched []string

	storeMu.RLock()
	for k := range store {
		if ok, _ := path.Match(pattern, k); ok {
			matched = append(matched, k)
		}
	}
	storeMu.RUnlock()

	if matched == nil {
		matched = []string{}
	}
	return Encode(matched, false)
}

// DBSIZE — returns the number of keys in the current database
func evalDBSIZE(args []string) []byte {
	return Encode(int64(StoreLen()), false)
}

// FLUSHDB — deletes all keys in the current database
func evalFLUSHDB(args []string) []byte {
	storeMu.Lock()
	store = make(map[string]*Obj)
	expires = make(map[*Obj]uint64)
	KeyspaceStat[0]["keys"] = 0
	storeMu.Unlock()
	return RESP_OK
}

// TYPE key — returns the type of the value stored at key
func evalTYPE(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'type' command"), false)
	}
	obj := Get(args[0])
	if obj == nil {
		return Encode("none", true)
	}
	return Encode(TypeName(obj.TypeEncoding), true)

}



func evalSET(args []string) []byte{
	//similarly for evalSET
	if len(args) <= 1{
		//iska mtlb we are not passing required arguemnts
		return Encode(errors.New("(error) ERR wrong number of arguments for 'set' commands"), false)
	}

	var key, value string
	var exDurationMs int64 = -1//as we know ki default value of expiration is -1

	key, value = args[0], args[1]
	otype,oEnc:=deduceTypeEncoding(value)

	for i := 2; i < len(args); i++{
		//as we are only implementing expiration as of now par SET functions implements a lot of other options too
		//since i got the key and value,everything else is just other
		switch args[i]{
		case "EX", "ex":
			//means users has passed some expiry
			//so we are doing i++ to know ki use r kya pass kiya hao
			i++
			if i == len(args){
				//mtlb suser kuch pass nhi kiya 
				return Encode(errors.New("(error) ERR syntax error"), false)
			}

			exDurationSec, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil{
				return Encode(errors.New("(error) ERR valye is not an integerr or out of range "), false)
			}
			exDurationMs = exDurationSec * 1000//because we are operation ms granuality
		default:
			return Encode(errors.New("(error) ERR syntax error"), false)
		}
	}

	//after this we have key,value set and it is ptional for expiration
	//so we will create a new Object
	Put(key, NewObj(value, exDurationMs, otype, oEnc))
	// c.Write([]byte("+OK\r\n"))
	// return nil
	return RESP_OK
	/*
	So instead of socket io we are just returning slice of byte, over the sokcet by eval and response 
	*/
}

func evalGET(args []string)[]byte{
	if len(args)!=1{
		return Encode(errors.New("(error) ERR wrong number of argments is passed bro"),false)
	}

	var key string=args[0]

	//get the key from the hash table 
	obj:=Get(key)//as we ahve seen humog hash table of string as key and value as object banaye the 

	//if key does not exist, return resp encoded nil
	if obj==nil{
		// c.Write(RESP_NIL)
		// return nil
		return RESP_NIL
	}

	//if key already expired then return nil
	// if obj.ExpiresAt!=-1 && obj.ExpiresAt<=time.Now().UnixMilli(){
	// 	// c.Write(RESP_NIL)
	// 	// return nil
	// 	return RESP_NIL
	// }

	//if key already expired then return nil
	if hasExpired(obj){
		return RESP_NIL
	}
	//return te RESP encoded value 
	// c.Write(Encode(obj.Value,false))
	// return nil
	return Encode(obj.Value,false)
}
//a nil is nothing but a string with -1 length
//so instead of writing again and again we just created constant object as RESP_NIL and referncing it everywhere


func evalTTL(args []string) []byte{
	if len(args)!=1{
		return Encode(errors.New("(error) ERR wrong number of argumenst for 'get' command"),false)
	}

	var key string=args[0]

	obj:=Get(key)

	//if the key does not exist, return RESP encoded -2 denoting key does not exist
	if obj==nil{
		// c.Write([] byte(":-2\r\n"))
		// return nil
		return RESP_MINUS_2
	}

	//if object exist, but no expiration is set on it then send -1
	// if obj.ExpiresAt==-1{  // Commented out - ExpiresAt field removed
	// 	// c.Write([]byte(":-1\r\n"))
	// 	// return nil
	// 	return RESP_MINUS_1
	// }

	//compute the time remaining for the key to expire and 
	//return the RESP encoded form of it 
	// durationMs:=obj.ExpiresAt-time.Now().UnixMilli()  // Commented out

	//if key expired i.e key does not exist hence return -2
	// if durationMs < 0{  // Commented out
	// 	// c.Write([]byte(":-2\r\n"))
	// 	// return nil
	// 	return RESP_MINUS_2
	// }

	exp, isExpirySet := getExpiry(obj)  // Fixed: handle both return values
	if !isExpirySet{  // Fixed: proper handling
		return RESP_MINUS_1
	}

	//if key expired i.e key does not exist hence return -2
	// PERF: use GlobalCachedTime instead of syscall
	if uint64(GlobalCachedTime)>uint64(exp){
		return RESP_MINUS_2
	}

	//compute the time remaining for the key to expire and 
	//return the RESP encoded form of it
	durationMs:=int64(exp)-GlobalCachedTime
	
	return Encode(durationMs/1000,false)
}
	// Duplicate code commented out below - was causing syntax error
	// durationMS:=exp-uint64(time.Now().UnixMilli())
	// c.Write(Encode(int64(durationMS/1000),false))
	// return nil
	// return Encode(int64(durationMs/1000), false)

// Continue with next function
func evalDEL(args []string) []byte{
	var countDeleted int = 0
	for _, key := range args{
		if ok := Del(key); ok{
			countDeleted++
		}
	}
	return Encode(countDeleted, false)
	// return nil
}

func evalEXPIRE(args []string) []byte{
	if len(args) <= 1{
		return Encode(errors.New("(error) ERR wrong number of arguments for 'expire' command"), false)
	}

	var key string = args[0]
	exDurationSec, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil{
		return Encode(errors.New("(error) ERR value is nt an integer or out of range"), false)
	}

	obj := Get(key)

	//0 if the timeout was not set: e.g Key doesn't exits, or operation skipped due to provided argument
	if obj == nil{
		// c.Write([]byte(":0\r\n")) //qki operation successful nhi hua
		// return nil
		return RESP_ZERO
	}

	// obj.ExpiresAt = time.Now().UnixMilli() + exDurationSec*1000

	setExpiry(obj,exDurationSec*1000)

	//1 print krenge if the timeout was set
	// c.Write([]byte(":1\r\n"))
	// return nil
	return RESP_ONE
}

//TODO: Make it async by forking a new process
func evalBGREWRITEAOF(args []string) []byte{
	DumpAllAOF()
	return RESP_OK
}

func evalINFO(args []string) []byte {
	var section string
	if len(args) > 0 {
		section = strings.ToLower(args[0])
	}

	var buf bytes.Buffer

	// Helper to add sections
	addSection := func(name string, addData func()) {
		if section == "" || section == name || section == "all" {
			buf.WriteString("# " + strings.ToUpper(name[0:1]) + name[1:] + "\r\n")
			addData()
			buf.WriteString("\r\n")
		}
	}

	addSection("server", func() {
		buf.WriteString("velox_version:1.0.0\r\n")
		buf.WriteString(fmt.Sprintf("uptime_in_seconds:%d\r\n", GetUptime()))
		buf.WriteString(fmt.Sprintf("uptime_in_days:%d\r\n", GetUptime()/86400))
		buf.WriteString("multiplexing_api:epoll\r\n")
		buf.WriteString("atomic_var_api:atomic-builtin\r\n")
	})

	addSection("clients", func() {
		// This would ideally count active client connections
		buf.WriteString(fmt.Sprintf("connected_clients:%d\r\n", atomic.LoadUint64(&GlobalMetrics.ConnectionsReceived)))
		buf.WriteString(fmt.Sprintf("pubsub_channels:%d\r\n", len(channels)))
	})

	addSection("memory", func() {
		// A rough estimate. In a real system we'd track allocation bytes via runtime.MemStats
		buf.WriteString(fmt.Sprintf("used_memory_human:%.2fM\r\n", float64(StoreLen()*128)/1024/1024))
		buf.WriteString(fmt.Sprintf("used_memory_dataset_perc:%.2f%%\r\n", 95.5))
	})

	addSection("stats", func() {
		buf.WriteString(fmt.Sprintf("total_connections_received:%d\r\n", atomic.LoadUint64(&GlobalMetrics.ConnectionsReceived)))
		buf.WriteString(fmt.Sprintf("total_commands_processed:%d\r\n", atomic.LoadUint64(&GlobalMetrics.CommandsProcessed)))
		buf.WriteString(fmt.Sprintf("instantaneous_ops_per_sec:%.2f\r\n", OpsPerSecond()))
	})

	addSection("keyspace", func() {
		for i := range KeyspaceStat {
			if KeyspaceStat[i]["keys"] > 0 {
				buf.WriteString(fmt.Sprintf("db%d:keys=%d,expires=0,avg_ttl=0\r\n", i, KeyspaceStat[i]["keys"]))
			}
		}
	})

	return Encode(buf.String(), false)
}

func evalCLIENT(args []string) []byte {
    if len(args) == 0 {
        return Encode("CLIENT subcommands: LIST, SETNAME, GETNAME", true)
    }
    sub := strings.ToUpper(args[0])
    switch sub {
    case "LIST":
        return Encode("id=1 addr=127.0.0.1:12345 fd=5 name=velox-cli age=0 idle=0 flags=N db=0 sub=0 psub=0 multi=-1 qbuf=0 qbuf-free=0 obl=0 oll=0 omem=0 events=r cmd=ping", false)
    case "SETNAME":
        if len(args) < 2 {
            return Encode(errors.New("ERR wrong number of arguments for 'CLIENT SETNAME'"), false)
        }
        return RESP_OK
    case "GETNAME":
        return Encode("velox-client", false)
    default:
        return Encode(errors.New("ERR unknown subcommand for CLIENT"), false)
    }
}

func evalLATENCY(args []string) []byte {
    if len(args) == 0 {
        return Encode("LATENCY subcommands: LATEST, RESET", true)
    }
    sub := strings.ToUpper(args[0])
    switch sub {
    case "LATEST":
        return Encode([]string{"[0, 0, 0, 0, 0]"}, false)
    case "RESET":
        return RESP_OK
    default:
        return Encode(errors.New("ERR unknown subcommand for LATENCY"), false)
    }
}

func evalSLEEP(args []string)[]byte{
	if len(args)!=1{
		return Encode(errors.New("ERR wrong number of arguments for 'SLEEP' command"),false)
	}

	durationSec,err:=strconv.ParseInt(args[0],10,64)
	if err!=nil{
		return Encode(errors.New("ERR value is not an integer or out of range"),false)
	}
	time.Sleep(time.Duration(durationSec)*time.Second)
	return RESP_OK
}

func evalMULTI(args []string)[]byte{
	return RESP_OK
}

// evalLRU returns diagnostics about the approximated LRU eviction system
// Shows: eviction strategy, sample size, pool size, top candidate idle time, keys count/limit
func evalLRU(args []string) []byte {
	poolSize, topIdle := ePool.PoolStats()

	var info []byte
	buf := bytes.NewBuffer(info)
	buf.WriteString(fmt.Sprintf("# Approximated LRU Stats\r\n"))
	buf.WriteString(fmt.Sprintf("eviction_strategy:%s\r\n", config.EvictionStrategy))
	buf.WriteString(fmt.Sprintf("lru_sample_size:%d\r\n", config.LRUSampleSize))
	buf.WriteString(fmt.Sprintf("eviction_pool_size:%d/%d\r\n", poolSize, 16))
	buf.WriteString(fmt.Sprintf("pool_top_idle_seconds:%d\r\n", topIdle))
	buf.WriteString(fmt.Sprintf("keys_count:%d\r\n", StoreLen()))
	buf.WriteString(fmt.Sprintf("keys_limit:%d\r\n", config.KeysLimit))
	buf.WriteString(fmt.Sprintf("eviction_ratio:%.2f\r\n", config.EvictionRatio))

	return Encode(buf.String(), false)
}

// evalOBJECT implements the OBJECT command (subset)
// Supports:
//   OBJECT IDLETIME <key> — returns idle time in seconds (how long since last access)
//   OBJECT HELP           — shows available subcommands
//   OBJECT ENCODING <key> — returns encoding type of the value
func evalOBJECT(args []string) []byte {
	if len(args) == 0 {
		return Encode(errors.New("ERR wrong number of arguments for 'OBJECT' command"), false)
	}

	sub := strings.ToUpper(args[0])
	switch sub {
	case "IDLETIME":
		// OBJECT IDLETIME <key> — returns how many seconds since the key was last accessed
		// This is the core debugging tool for approximated LRU
		if len(args) < 2 {
			return Encode(errors.New("ERR wrong number of arguments for 'OBJECT IDLETIME'"), false)
		}
		idleTime, exists := GetKeyIdleTime(args[1])
		if !exists {
			return RESP_NIL
		}
		return Encode(int64(idleTime), false)

	case "ENCODING":
		// OBJECT ENCODING <key> — returns the internal encoding of the value
		if len(args) < 2 {
			return Encode(errors.New("ERR wrong number of arguments for 'OBJECT ENCODING'"), false)
		}
		obj := Get(args[1])
		if obj == nil {
			return RESP_NIL
		}
		return Encode(EncodingName(obj.TypeEncoding), false)

	case "HELP":
		return Encode("OBJECT subcommands: IDLETIME <key>, ENCODING <key>, HELP", false)

	default:
		return Encode(errors.New("ERR unknown subcommand for OBJECT"), false)
	}
}


func executeCommand(cmd *RedisCmd, c *Client) []byte{
	TrackCommand()
	//It's job is like depending on what job is sent to us
	//we trigger the corresponding eval function
		switch cmd.Cmd{
		case "PING":
			return evalPING(cmd.Args)
		case "SET":
			return evalSET(cmd.Args)
		case "GET":
			return evalGET(cmd.Args)
		case "TTL":
			return evalTTL(cmd.Args)
		case "DEL":
			return evalDEL(cmd.Args)
		case "EXPIRE":
			return evalEXPIRE(cmd.Args)
		case "BGREWRITEAOF":
			return evalBGREWRITEAOF(cmd.Args)
		case "INCR":
			return evalINCR(cmd.Args)
		case "DECR":
			return evalDECR(cmd.Args)
		case "INCRBY":
			return evalINCRBY(cmd.Args)
		case "DECRBY":
			return evalDECRBY(cmd.Args)
		case "MSET":
			return evalMSET(cmd.Args)
		case "MGET":
			return evalMGET(cmd.Args)
		case "INFO":
			return evalINFO(cmd.Args)
		case "CLIENT":  
			return evalCLIENT(cmd.Args)
		case "LATENCY": 
			return evalLATENCY(cmd.Args)
		case "LRU":
			return evalLRU(cmd.Args)
		case "OBJECT":
			return evalOBJECT(cmd.Args)
		case "SLEEP":
			return evalSLEEP(cmd.Args)
		case "MULTI":
			c.TxnBegin()
			return evalMULTI(cmd.Args)
		case "EXEC":
			if !c.isTxn{
				return Encode(errors.New("ERR EXEC without MULTI"),false)
			}
			return c.TxnExec()
		case "DISCARD":
			if !c.isTxn{
				return Encode(errors.New("ERR DISCARD without MULTI"),false)
			}
			c.TxnDiscard()
			return RESP_OK
		case "COMMAND":
			return Encode("OK", true)
		// --- Utility commands ---
		case "EXISTS":
			return evalEXISTS(cmd.Args)
		case "KEYS":
			return evalKEYS(cmd.Args)
		case "DBSIZE":
			return evalDBSIZE(cmd.Args)
		case "FLUSHDB":
			return evalFLUSHDB(cmd.Args)
		case "TYPE":
			return evalTYPE(cmd.Args)
		// --- Hash commands ---
		case "HSET":
			return evalHSET(cmd.Args)
		case "HGET":
			return evalHGET(cmd.Args)
		case "HGETALL":
			return evalHGETALL(cmd.Args)
		case "HDEL":
			return evalHDEL(cmd.Args)
		case "HLEN":
			return evalHLEN(cmd.Args)
		case "HEXISTS":
			return evalHEXISTS(cmd.Args)
		case "HKEYS":
			return evalHKEYS(cmd.Args)
		case "HVALS":
			return evalHVALS(cmd.Args)
		case "HINCRBY":
			return evalHINCRBY(cmd.Args)
		case "HSETNX":
			return evalHSETNX(cmd.Args)
		// --- List commands ---
		case "LPUSH":
			return evalLPUSH(cmd.Args)
		case "RPUSH":
			return evalRPUSH(cmd.Args)
		case "LPOP":
			return evalLPOP(cmd.Args)
		case "RPOP":
			return evalRPOP(cmd.Args)
		case "LRANGE":
			return evalLRANGE(cmd.Args)
		case "LLEN":
			return evalLLEN(cmd.Args)
		case "LINDEX":
			return evalLINDEX(cmd.Args)
		// --- Set commands ---
		case "SADD":
			return evalSADD(cmd.Args)
		case "SREM":
			return evalSREM(cmd.Args)
		case "SMEMBERS":
			return evalSMEMBERS(cmd.Args)
		case "SISMEMBER":
			return evalSISMEMBER(cmd.Args)
		case "SCARD":
			return evalSCARD(cmd.Args)
		case "SINTER":
			return evalSINTER(cmd.Args)
		case "SUNION":
			return evalSUNION(cmd.Args)
		// --- ZSet commands ---
		case "ZADD":
			return evalZADD(cmd.Args)
		case "ZSCORE":
			return evalZSCORE(cmd.Args)
		case "ZRANK":
			return evalZRANK(cmd.Args)
		case "ZRANGE":
			return evalZRANGE(cmd.Args)
		case "ZCARD":
			return evalZCARD(cmd.Args)
		case "ZREM":
			return evalZREM(cmd.Args)
		// --- Pub/Sub commands ---
		case "SUBSCRIBE":
			return pubsubSubscribe(c, cmd.Args)
		case "UNSUBSCRIBE":
			return pubsubUnsubscribe(c, cmd.Args)
		case "PUBLISH":
			return pubsubPublish(cmd.Args)
		default:
			return Encode(errors.New("ERR unknown command '"+cmd.Cmd+"'"), false)
		}
}

func executeCommandToBuffer(cmd *RedisCmd,buf *bytes.Buffer,c *Client){
	buf.Write(executeCommand(cmd,c))
}

// ExecuteCommandPublic is the exported version of executeCommand for use by the server package
func ExecuteCommandPublic(cmd *RedisCmd, c *Client) []byte {
	return executeCommand(cmd, c)
}

// func EvalAndRespond(cmd *Rediscmd,c net.Conn)error{
/*
EvalAndRespond processes a batch of commands and writes all responses back to the client.

PERFORMANCE FIX: Buffer reuse

BEFORE:
  var response []byte
  buf := bytes.NewBuffer(response)

  This created a NEW bytes.Buffer on every single EvalAndRespond call.
  bytes.NewBuffer allocates:
    1. A Buffer struct (on the heap)
    2. An internal []byte slice (initial capacity = len(response) = 0)
  Then as we write responses, the internal slice grows (via append), causing
  MORE allocations as it doubles in size: 0 → 64 → 128 → 256 → ...
  At 100K QPS = 100K Buffer structs + 300K+ internal slice growths = 400K+ allocations/sec

AFTER:
  buf := c.WriteBuf
  buf.Reset()

  Uses the client's pre-allocated WriteBuf. Reset() sets the length to 0 but
  KEEPS the underlying byte slice at its current capacity. After the first few
  requests, the buffer grows to the typical response size and stays there.
  
  Example: if most responses are ~200 bytes, after the first request the buffer
  has 200+ bytes of capacity. Every subsequent request reuses that capacity.
  Result: ZERO allocations for the buffer after warmup.
*/
func EvalAndRespond(cmds []*RedisCmd, c *Client){
	buf := WriteBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer WriteBufPool.Put(buf)
	//our logic didn't changed, but the way we are consuming has changed
	for _, cmd := range cmds{
		//if txn is not in progress , then we ca simply 
		//execute the command and add the response to the buffer
		if !c.isTxn{
			executeCommandToBuffer(cmd,buf,c)
			continue
		}

		//if the txn is in progress, we enqueue the command 
		//and add the queued response to the buffer
		if !txnCommands[cmd.Cmd]{
			//if the command is queuabe the enqueue
			c.TxnQueue(cmd)
			buf.Write(RESP_QUEUED)
		}else{
			//if txn is active and the command is non-queuable 
			//ex: EXEC,DISCARD
			//we execute the command and gather the response in buffer
			executeCommandToBuffer(cmd,buf,c)
		}
	}
	// _, err := c.Write(buf.Bytes())
	// return err
	c.Write(buf.Bytes())
}

