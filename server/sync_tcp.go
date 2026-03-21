
func readCommand(c net.Conn)(string,error){
	/*
	Take the socket connection and basically fire the system call Read

	TODO: Max read in one shot is 512 bytes
	To allow input>512 bytes , then repreated read until
	we get EOF or designated delimeter

	It is listening over the socket and it is trying to read messag eover the socket 
	if there is nothing taht is coming from my lcient then it is a blocking call, until i get somehting from client 
	when we read it we put it into buffer and then, we get the number of bytes , if there is error we throw errer else we send it back 
	*/
	var buf []byte=make([]byte,512)
	n,err:=c.Read(buf[:])
	if err!=nul{
		return "",err
	}
	return string(buf[:n]),nil
}

func RunSyncTCPServer(){


	log.Println("startign a synchronous TCP Server on", config.Host,config.Port)

	var con_client int=0;
	//this will hold the number of concurrent client that are connceted at the moment
	/*
	It is just some extra things like we want to know that yes we have this much m=concurrent server
	*/

	//listening to the configured host:port
	/*
	Our server will start listening to the port that means any of the client can talk to server from the port upon which it is listening to
	once  our server is started then i will run an infinite loop like you can see below
	*/
	lsnr,err:=net.Listen("top",config.Host+":"+strconv.Item(config.Port))
	if(err!=nil){
		panic(err)
	}

	for{
		/*
		This Infinite for loop is to wait for infinite conncetion to get connected, so now any client can be able to connnect to server 
		for us to tell that  hey i am waiting for a new conncetion to be connected so we are doing  this blocking call 
		as soon as the client is connected we will move forward ele wr will thrown an error
		*/
		c,err:=lsnr.Accept()
		if err:=nil{
			panic(err)
		}
		//incrementing the number of concurrent clients 
		con_client+=1
		log.Println("Client connected with address: ", c.RemoteAddr(), "concurrent clients", con_client)

		/*
		Another infinite loop for 
		we want our clients to continuously sends us command like put this key,get this key etc
		*/
		for{
			//over the docket, continuously read the command and print it out 
			cmd,err:=readCommand(c)

		}
	}
}
