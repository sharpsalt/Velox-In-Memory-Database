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