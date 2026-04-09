import (
	"errors"
	"io"
	"strconv"
	"time"
)

var RESP_NIL []byte=[]byte("$-1\r\n")

// func evalPING(args []string,c net.Conn)error{
func evalPING(args []string,c io.ReadWriter) error{
	var b []byte 

	if len(args)>=2{
		//means if the redis cli passes us more than 1 arguments then this will invoke 
		return errors.New("ERR wrong number of arguments for 'ping' command")
	}

	if len(args)==0{
		//we will encode it into RESP
		//encode function is to take the raw type and convert it to another encoded resp format
		//Because server has to respond in resp format so the server will also do get the thing 
		b=Encode("PONG",true)
	}else{
		b=Encode(args[0],false)
	}

	_,err:=c.Write(b)
	return err 
}

func evalSET(args []string, io.ReadWriter) error{
	if len(args)<=1{
		//iska mtlb we are not passing required arguemnts
		return errors.New("(error) ERR wrong number of arguments for 'set' commands")
	}

	var key,value string
	var exDurationMs int64=-1//as we know ki default value of expiration is -1

	key,value=args[0],args[1]

	for i:=2;i<len(args);i++{
		//as we are only implementing expiration as of now par SET functions implements a lot of other options too
		//since i got the key and value,everything else is just other
		switch args[i]{
		case: "EX","ex":
			//means users has passed some expiry
			//so we are doing i++ to know ki use r kya pass kiya hao
			i++;
			if i==len(args){
				//mtlb suser kuch pass nhi kiya 
				return errors.New("(error) ERR syntax error")
			}

			exDurationSec,err:=strconv.ParseInt(args[3],10,64)
			if err!=nil{
				return errors.New("(error) ERR valye is not an integerr or out of range ")
			}
			exDurationMs=exDurationSec*1000//because we are operation ms granuality
		default:
			return errors.New("(error) ERR syntax error")
		}
	}

	//after this we have key,value set and it is ptional for expiration
	//so we will create a new Object
	Put(key,NewObj(value,exDurationMs))
	c.Write([]byte("+OK\r\n"))
	return nil
}

func evalGET(args []string,c io.ReadWriter) error{
	if len(args)!=1{
		return errors.New("(error) ERR wrong number of argments is passed bro")
	}

	var key string=args[0]

	//get the key from the hash table 
	obj:=Get(key)//as we ahve seen humog hash table of string as key and value as object banaye the 

	//if key does not exist, return resp encoded nil
	if obj==nil{
		c.Write(RESP_NIL)
		return nil
	}

	//if key already expired then return nil
	if obj.ExpiresAt!=-1 && obj.ExpiresAt<=time.Now().UnixMilli(){
		c.Write(RESP_NIL)
		return nil
	}


	//return te RESP encoded value 
	c.Write(Encode(obj.Value,false))
	return nil
}
//a nil is nothing but a string with -1 length
//so instead of writing again and again we just created constant object as RESP_NIL and referncing it everywhere

// func EvalAndRespond(cmd *Rediscmd,c net.Conn)error{
func EvalAndRespond(cmd *RedisCmd, c io.ReadWriter) error{
	//It's job is like depending on what job is sent to us
	//we trigger the corresponding eval function

	switch cmd.Cmd{
	case "PING":
		return evalPING(cmd.Args,c)
	case "SET":
		return evalSET(cmd.Args,c)
	case "GET":
		return evalGET(cmd.Args,c)
	case "TTL":
		return evalTTL(cmd.Args,c)
	default:
		return evalPING(cmd.Args,c);
	}
}