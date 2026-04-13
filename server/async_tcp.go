package server

import(
	"log"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/sharpsalt/Velox-In-Memory-Database/core"
)

var con_client int=0
var cronFrequency time.Duration=1*time.Second //a cron frequency of 1s
var lastCronExecTime time.Time=time.Now() //and we are maintaining, last time it ran

// readCommands reads RESP commands from a file descriptor and returns them
func readCommands(c core.FDComm)([]*core.RedisCmd,error){
	/*
	Take the socket connection and basically fire the system call Read
	It is listening over the socket and it is trying to read message over the socket 
	if there is nothing that is coming from my client then it is a blocking call, until i get something from client 
	when we read it we put it into buffer and then, we get the number of bytes , if there is error we throw error else we send it back 
	*/
	var buf []byte=make([]byte,512)
	n,err:=c.Read(buf[:])//reading it in buffer from Client
	if err!=nil{
		return nil, err
	}
	/*
	Pipelining-> multiple commands
	like earlier we used to decode once, but now we are continuously decoding it
	the idea here is we would want to accept multiple commands 
	so we want commands back to back literally concatenated
	*/
	values, err := core.Decode(buf[:n])
	if err != nil{
		return nil, err
	}

	var cmds []*core.RedisCmd = make([]*core.RedisCmd, 0)
	for _, value := range values{
		tokens, err := core.ToArrayString(value.([]interface{}))
		if err != nil{
			return nil, err
		}
		//so here we are creating redis command objects 
		cmds = append(cmds, &core.RedisCmd{
			Cmd:  strings.ToUpper(tokens[0]),
			Args: tokens[1:],
		})
	}
	return cmds, nil
}

func RunAsyncTCPServer(cfg *Config) error{
	log.Println("Starting an asynchronous TCP Server on", cfg.Host, cfg.Port)
	// since humlog linux based system use krrhe hai so 
	// so we are using epoll
	max_clients := 20000

	//create EPOLL Event Objects to hold events 
	/*
	This EPOLL event , like we ahve to register our file descriptor in EPOLL
	but when some file descriptor is ready for IO , we will get it in I/O
	This buffer will be holding file descriptor at a ready by EPOLL system call

	this events is for holding those events which are ready
	*/
	var events []syscall.EpollEvent = make([]syscall.EpollEvent, max_clients)

	//Create a socket
	/*
	here we do raw sokcet handling, we can't use raw abstaction as we are handling raw file descriptors

	so a normal socket which is an IPv4, which is NON_BLOCKING sokcet_stream

	socket_stream: i want to keep this TCP connection as soon as i got reply, basically i want to keep this connection open
	*/
	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM | syscall.O_NONBLOCK, 0)
	if err != nil{
		return err
	}
	//one the connection is done please close the sockey like i internally know how does the defer works internally so ussi hisaab se
	defer syscall.Close(serverFD)

	//Set the Socket operate in a non blocking mode
	/*
	Because we want ServerFD to get monitored so
	*/
	if err=syscall.SetNonblock(serverFD,true);err!=nil{
		return err
	}

	//Bind the IP and the port
	ip4 := net.ParseIP(cfg.Host).To4()
	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: cfg.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil{
		return err
	}

	//Then we will listen, like we start to listen as FD is binded to port etc
	if err = syscall.Listen(serverFD, max_clients); err != nil{
		return err
	}


	//Async IO Starts

	/*
	We create an EPOLL
	epoll itslef has a file descriptor of its own
	age of every fd is different inorder to identify that
	*/
	epollFD, err := syscall.EpollCreate1(0)
	if err != nil{
		log.Fatal(err)
	}
	defer syscall.Close(epollFD)

	/*
	you want the server (FD) to get monitored as well so we want the server to be monitored
	so we are creating an EPOLL event so isme hum kya kya montor krrhe dkeho 
	like if anytime an incoming request is coming to my server , hen basically notify me 
	so using EPOLLFD adds that to be monitored

	*/
	var socketServerEvent syscall.EpollEvent = syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:    int32(serverFD),
	}

	//listen to read evenets on the server itself 
	if err = syscall.EpollCtl(epollFD, syscall.EPOLL_CTL_ADD, serverFD, &socketServerEvent); err != nil{
		return err
	}

	for{
		/*
		The first thing which we do to execute this cron

		now that my time, means when the loops execute and control flow comes over here and if enough time has passed then other

		we are just plugging our logic as first
		everytime the control flow comes here then each time ye run krega


		Redis uses interrupst, kernel level interrupts and then it would run it 

		sine we are just understanding how single threaded high performance data can be build
		*/
		if time.Now().After(lastCronExecTime.Add(cronFrequency)){
			core.DeleteExpiredKey()
			lastCronExecTime = time.Now()
		}



		//basically we have to constantly monitor if an IO is ready
		//that why i am invoking EPOLL wait

		//see if any FD is ready for an IO
		nevents, e := syscall.EpollWait(epollFD, events[:], -1)
		//EpollWait will monitor if any IO is ready, and put it in buffer(evenets buffer), if none is there then the call wouldget blocked

		if e != nil{
			continue
		}

		for i := 0; i < nevents; i++{
			//if the socket server itself is ready for an IO
			if int(events[i].Fd) == serverFD{//each events has a File descriptors
				//if my server is ready for IO then i have to accept the incoming skcet
				//accept the incoming connection from a client
				fd, _, err := syscall.Accept(serverFD) //you accept and get the fd between server and the new client which is getted connected
				if err != nil{
					log.Println("err", err)
					continue
				}

				//increase the number of concurrent clients count
				con_client++
				syscall.SetNonblock(fd, true)

				//add this new TCP connetion to be monitored
				var socketClientEvent syscall.EpollEvent = syscall.EpollEvent{
					Events: syscall.EPOLLIN,
					Fd:    int32(fd),
				}
				if err := syscall.EpollCtl(epollFD, syscall.EPOLL_CTL_ADD, fd, &socketClientEvent); err != nil{
					log.Fatal(err)
				}
			}else{
				//if it is not my server which means that some client that is already connected to the server, then do somthing
				comm := core.FDComm{Fd: int(events[i].Fd)}
				cmds, err := readCommands(comm) //instead of passing 1 command, we will pass many commands
				if err != nil{
					syscall.Close(int(events[i].Fd))
					con_client -= 1
					continue
				}
				respond(cmds, comm)
			}
		}
	}
}

/*
In synchronous we will get abstracted socket connection
while uspe hum asynchronous me fd use krte hai 



The only place wehre i can put my autodelete is inside evntloop mtlb yaha pe, as in Redis the deletion happens in two phases->active and passive
so the event loop that we wrote,basically the infinite for loop that is waiting on epoll wait,just before we added
*/
