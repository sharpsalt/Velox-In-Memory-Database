package core

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

//TODO: Support sync write
func dumpKey(w *bufio.Writer, key string, obj *Obj) {
	t := getType(obj.TypeEncoding)
	enc := getEncoding(obj.TypeEncoding)

	switch t {
	case OBJ_TYPE_STRING:
		cmd := fmt.Sprintf("SET %s %s", key, obj.Value)
		tokens := strings.Split(cmd, " ")
		w.Write(Encode(tokens, false))

	case OBJ_TYPE_HASH:
		// Dump as: HSET key field1 val1 field2 val2 ...
		var fields []string
		switch enc {
		case OBJ_ENCODING_ZIPLIST:
			zl := obj.Value.(*ZipList)
			fields = zl.HashGetAll()
		case OBJ_ENCODING_HT:
			ht := obj.Value.(map[string]string)
			for k, v := range ht {
				fields = append(fields, k, v)
			}
		}
		if len(fields) > 0 {
			tokens := append([]string{"HSET", key}, fields...)
			w.Write(Encode(tokens, false))
		}

	case OBJ_TYPE_LIST:
		// Dump as: RPUSH key elem1 elem2 ...
		ql := obj.Value.(*QuickList)
		elems := ql.AllElements()
		if len(elems) > 0 {
			tokens := append([]string{"RPUSH", key}, elems...)
			w.Write(Encode(tokens, false))
		}

	case OBJ_TYPE_SET:
		// Dump as: SADD key member1 member2 ...
		var members []string
		switch enc {
		case OBJ_ENCODING_INTSET:
			is := obj.Value.(*IntSet)
			for _, v := range is.Members() {
				members = append(members, strconv.FormatInt(v, 10))
			}
		case OBJ_ENCODING_HT:
			ht := obj.Value.(map[string]struct{})
			for k := range ht {
				members = append(members, k)
			}
		}
		if len(members) > 0 {
			tokens := append([]string{"SADD", key}, members...)
			w.Write(Encode(tokens, false))
		}
	}
}


//TODO: To to new and switch
func DumpAllAOF(){
	fp,err:=os.OpenFile(config.AOFFile,os.O_CREATE|os.O_WRONLY|os.O_TRUNC,0644)
	if err!=nil{
		fmt.Println("error",err)
		return
	}
	defer fp.Close() // #2 FIX: Close the file to prevent file descriptor leaks

	writer := bufio.NewWriter(fp) // Buffered writes for better I/O performance

	log.Println("rewriting AOF File at ",config.AOFFile)

	// Acquire read lock while iterating the store
	storeMu.RLock()
	for k,obj:=range store{
		dumpKey(writer,k,obj) //While dumping AOF File we will keep it simple like go through all of key and dump in aodf format
	}
	storeMu.RUnlock()

	writer.Flush() // Flush any remaining buffered data

	log.Println("AOF File rewrite complete")
}

// ReplayAOF reads the AOF file on startup and replays all persisted commands
// This restores the database state from the last dump
func ReplayAOF(){
	fp,err:=os.Open(config.AOFFile)
	if err!=nil{
		// File doesn't exist on first run — that's fine
		log.Println("no AOF file found at",config.AOFFile,"— starting with empty database")
		return
	}
	defer fp.Close()

	// Read the entire file content
	stat,err := fp.Stat()
	if err != nil {
		log.Println("error stating AOF file:", err)
		return
	}
	if stat.Size() == 0 {
		log.Println("AOF file is empty — starting with empty database")
		return
	}

	buf := make([]byte, stat.Size())
	n, err := fp.Read(buf)
	if err != nil {
		log.Println("error reading AOF file:", err)
		return
	}

	// Decode all the RESP-encoded commands from the file
	values, err := Decode(buf[:n])
	if err != nil {
		log.Println("error decoding AOF file:", err)
		return
	}

	var replayedCount int
	for _, value := range values {
		// Each value should be an array (a RESP command)
		arr, ok := value.([]interface{})
		if !ok {
			continue
		}
		tokens := make([]string, len(arr))
		for i, v := range arr {
			tokens[i] = fmt.Sprintf("%v", v)
		}
		if len(tokens) == 0 {
			continue
		}

		// Replay supported commands
		cmd := strings.ToUpper(tokens[0])
		switch cmd {
		case "SET":
			if len(tokens) >= 3 {
				evalSET(tokens[1:])
				replayedCount++
			}
		case "HSET":
			if len(tokens) >= 4 {
				evalHSET(tokens[1:])
				replayedCount++
			}
		case "RPUSH":
			if len(tokens) >= 3 {
				evalRPUSH(tokens[1:])
				replayedCount++
			}
		case "SADD":
			if len(tokens) >= 3 {
				evalSADD(tokens[1:])
				replayedCount++
			}
		}
	}

	log.Printf("AOF replay complete: restored %d commands from %s\n", replayedCount, config.AOFFile)
}
