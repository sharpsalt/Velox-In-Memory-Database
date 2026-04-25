package core

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

//TODO: Support non-kv data structure
//TODO: Support sync write
func dumpKey(w *bufio.Writer,key string,obj *Obj){
	cmd:=fmt.Sprintf("SET %s %s", key,obj.Value)
	tokens:=strings.Split(cmd," ")
	w.Write(Encode(tokens,false))
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

// ReplayAOF reads the AOF file on startup and replays all persisted SET commands
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

		// We only replay SET commands for now
		cmd := strings.ToUpper(tokens[0])
		switch cmd {
		case "SET":
			if len(tokens) >= 3 {
				evalSET(tokens[1:])
				replayedCount++
			}
		}
	}

	log.Printf("AOF replay complete: restored %d keys from %s\n", replayedCount, config.AOFFile)
}