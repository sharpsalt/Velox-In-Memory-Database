package server

import(
"bytes"
"fmt"
"io"
"log"
"net"
"strconv"
"strings"

"github.com/sharpsalt/Velox-In-Memory-Database/core"
)

type Config struct{
	Host string
	Port int
}

func readCommand(c net.Conn) ([]*core.RedisCmd, error) {
	/*
	Take the socket connection and basically fire the system call Read

	TODO: Max read in one shot is 512 bytes
	To allow input>512 bytes , then repreated read until
	we get EOF or designated delimeter

	It is listening over the socket and it is trying to read messag eover the socket 
	if there is nothing taht is coming from my lcient then it is a blocking call, until i get somehting from client 
	when we read it we put it into buffer and then, we get the number of bytes , if there is error we throw errer else we send it back 
	*/
	var buf []byte = make([]byte, 64*1024) // #3 FIX: 64KB buffer instead of 512 bytes
	n, err := c.Read(buf[:])//reading it in buffer from Client
	if err != nil{
		return nil, err
	}
	/*
	Pipelining-> me ka
	     like earlier we used to decode once, but ab we are continuously decoding it
		 the idea here is we would want to accept multiple commands 
		 so we want commands back to back literally concatenated

	*/
	values,err:=core.Decode(buf[:n])
	if err!=nil{
		return nil,err
	}
	/*
	and decoding it into an array of strings
	when any redis cleint wants to issue a command to redis server then a command typically has 
	root command and argument,all of them is sent to as an array of strings so if  am doing PUT key,value-> then it will sends as array of strings 
	*/
	// tokens,err:=core.DecodeArrayString(buf[:n])
	// if err!=nil{
	// 	return nil,err
	// }
	// // return val.(*core.Command), nil
	// //once we had tokens we are creating redis command object and passing the root command in which tokens[0] will become command and rest will become the argument 
	// return &core.Rediscmd{
	// 	Cmd:strings.ToUpper(tokens[0]),
	// 	Args: tokens[1:],
	// },nil

	var cmds []*core.RedisCmd = make([]*core.RedisCmd, 0)
	for _, value := range values{
		tokens, err := core.ToArrayString(value.([]interface{}))
		if err != nil{
			return nil, err
		}
		//so here are we 
		cmds = append(cmds, &core.RedisCmd{
			Cmd:  strings.ToUpper(tokens[0]),
			Args: tokens[1:],
		})
	}
	return cmds, nil
}

func respondError(err error, c io.ReadWriter){
	//It write on TCP socket of stream of bytes where we are dping stream formatting 
	//we know that whenever we send an error over TCP connection to the Redis cli, it needs to be encoded
	//so it starts with - sign,then error string
	//string is converted to byte and responding back to client

	c.Write([]byte(fmt.Sprintf("-%s\r\n", err)))
}

func respondSync(cmds []*core.RedisCmd, c net.Conn){//Responding with connection
	//we passed give the command and given the socket connection, just writing it back over the socket
	//like whatever we got we are sending it back to the client

	// Wrap net.Conn as a Client for EvalAndRespond
	// For sync server, we create a temporary client wrapper
	client := &core.Client{Fd: -1} // Fd=-1 since we use net.Conn's Write
	// We can't directly use net.Conn with EvalAndRespond which expects *Client
	// So we handle it manually
	var response []byte
	buf := &bytes.Buffer{}
	for _, cmd := range cmds {
		buf.Write(core.ExecuteCommandPublic(cmd, client))
	}
	response = buf.Bytes()
	_, err := c.Write(response)
	if err != nil {
		respondError(err, c)
	}
	/*
	Basically we are building an echo server like whatever we are getting from client, we are sending it back to him
	*/
}

func RunSyncTCPServer(cfg *Config){

	log.Println("startign a synchronous TCP Server on", cfg.Host, cfg.Port)
	var con_client int = 0
	//this will hold the number of concurrent client that are connceted at the moment
	/*
	It is just some extra things like we want to know that yes we have this much m=concurrent server
	*/

	//listening to the configured host:port
	/*
	Our server will start listening to the port that means any of the client can talk to server from the port upon which it is listening to
	once  our server is started then i will run an infinite loop like you can see below
	*/
	lsnr, err := net.Listen("tcp", cfg.Host+":"+strconv.Itoa(cfg.Port))
	if err != nil{
		panic(err)
	}
	for{
		/*
		This Infinite for loop is to wait for infinite conncetion to get connected, so now any client can be able to connnect to server
		for us to tell that  hey i am waiting for a new conncetion to be connected so we are doing  this blocking call
		as soon as the client is connected we will move forward ele wr will thrown an error
		*/
		c, err := lsnr.Accept()
		if err != nil{
			panic(err)
		}
		//incrementing the number of concurrent clients
		con_client += 1
		log.Println("Client connected with address: ", c.RemoteAddr(), "concurrent clients", con_client)
		/*
		Another infinite loop for
		we want our clients to continuously sends us command like put this key,get this key etc
		*/
		for{
			//over the docket, continuously read the command and print it out
			cmds, err := readCommand(c)
			/*
			as the read command is done, this connect the connection else if the error is propogated back (like client is dissconneted), then err!=null
			then at time i will close my socket connection, like i want to reduce the number of concurrent client whihc i am handling
			and print krenge like earlier i have this much now i have these much
			and if it is graceful termination, like where the client is sending th termination, i am simply breaking out of loop

			and if my error is not nil then i wills imply tekk ok ye hai
			*/
			if err != nil{
				c.Close()
				con_client -= 1
				log.Println("client disconnected", c.RemoteAddr(), "concurrent clients", con_client)
				if err == io.EOF{
					break
				}
					log.Println("error reading command", err)
				continue
			}
			log.Println("command", cmds)
			respondSync(cmds, c)
		}
	}
}
/*
Since when you run it, you'll see that we can;t add more than 1 like even if we add then it will simply won't acknowledge us
Now why does it happen
because our server is single threaded, we have for loop inside for loop. so until our client disconnects 
then your 2nd client will get the chance 


So what actually happen when we do connect a redis client to the server
redis server bhi to bhai TCP hi hoga so, so as sson as redis client is connected some message will exchange
like jba connect hi krenge by 
./src/redis-cli -p 7379
thnn ye backend me mtlb dusre terminal pe jaise hi connect hoga to 
command *1 
$7
COMMAND

Now you might aks that what it actually is: so this is what redis serialization protocol is all about
this is the command redis-cli sends to server when connection got established


*/
