package core

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"time"
)

var RESP_NIL []byte=[]byte("$-1\r\n")
var RESP_OK []byte=[]byte("+OK\r\n")
var RESP_ZERO []byte=[]byte(":0\r\n")
var RESP_ONE []byte=[]byte(":1\r\n")
var RESP_MINUS_1 []byte=[]byte(":-1\r\n")
var RESP_MINUS_2 []byte=[]byte(":-2\r\n")

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

func evalSET(args []string) []byte{
	//similarly for evalSET
	if len(args) <= 1{
		//iska mtlb we are not passing required arguemnts
		return Encode(errors.New("(error) ERR wrong number of arguments for 'set' commands"), false)
	}

	var key, value string
	var exDurationMs int64 = -1//as we know ki default value of expiration is -1

	key, value = args[0], args[1]

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
	Put(key, NewObj(value, exDurationMs))
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
	if obj.ExpiresAt!=-1 && obj.ExpiresAt<=time.Now().UnixMilli(){
		// c.Write(RESP_NIL)
		// return nil
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
	if obj.ExpiresAt==-1{
		// c.Write([]byte(":-1\r\n"))
		// return nil
		return RESP_MINUS_1
	}

	//compute the time remaining for the key to expire and 
	//return the RESP encoded form of it 
	durationMs:=obj.ExpiresAt-time.Now().UnixMilli()

	//if key expired i.e key does not exist hence return -2
	if durationMs < 0{
		// c.Write([]byte(":-2\r\n"))
		// return nil
		return RESP_MINUS_2
	}

	// c.Write(Encode(int64(durationMS/1000),false))
	// return nil
	return Encode(int64(durationMs/1000), false)
}

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

	obj.ExpiresAt = time.Now().UnixMilli() + exDurationSec*1000

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

// func EvalAndRespond(cmd *Rediscmd,c net.Conn)error{
func EvalAndRespond(cmds []*RedisCmd, c io.ReadWriter) error{
	//It's job is like depending on what job is sent to us
	//we trigger the corresponding eval function

	var response []byte
	buf := bytes.NewBuffer(response) // this is where we are buffering all 
	//our logic didn't chnaged, but the way we are consuming has changed
	for _, cmd := range cmds{
		switch cmd.Cmd{
		case "PING":
			buf.Write(evalPING(cmd.Args))
		case "SET":
			buf.Write(evalSET(cmd.Args))
		case "GET":
			buf.Write(evalGET(cmd.Args))
		case "TTL":
			buf.Write(evalTTL(cmd.Args))
		case "DEL":
			buf.Write(evalDEL(cmd.Args))
		case "EXPIRE":
			buf.Write(evalEXPIRE(cmd.Args))
		case "BGREWRITEAOF":
			buf.Write(evalBGREWRITEAOF(cmd.Args))
		default:
			buf.Write(evalPING(cmd.Args))
		}
		/*
		Earlier we used to return and like we used to pass io.ReadWriter but now instead of that the eval function that we ahev si returning the output
		but here we are putting it in buffer
		*/
	}
	_, err := c.Write(buf.Bytes())
	return err
}
