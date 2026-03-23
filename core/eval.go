
func evalPING(args []string,c net.Conn)error{
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

func EvalAndRespond(cmd *Rediscmd,c net.Conn)error{
	//It's job is like depending on what job is sent to us
	//we trigger the corresponding eval function

	switch cmd.Cmd{
	case "PING":
		return evalPING(cmd.Args,c)
	default:
		return evalPING(cmd.Args,c);
	}
}