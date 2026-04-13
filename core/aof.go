package core

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

//TODO: Support non-kv data structure
//TODO: Support sync write
func dumpKey(fp *os.File,key string,obj *Obj){
	cmd:=fmt.Sprintf("SET %s %s", key,obj.Value)
	tokens:=strings.Split(cmd,"")
	fp.Write(Encode(tokens,false))
}


//TODO: To to new and switch
func DumpAllAOF(){
	fp,err:=os.OpenFile(config.AOFile,os.O_CREATE|os.O_WRONLY,0644)
	if err!=nil{
		fmt.Println("error",err)
		return
	}
	log.Println("rewriting AOF File at ",config.AOFile)
	for k,obj:=range store{
		dumpKey(fp,k,obj) //While dumping AOF File we will keep it simple like go through all of key and dump in aodf format
	}
	log.Println("AOF File rewrite complete")
}