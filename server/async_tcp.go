package server

import(
"log"
"net"
"syscall"

"github.com/sharpsalt/Velox-In-Memory-Database/core"
"github.com/sharpsalt/Velox-In-Memory-Database/config"
)

con_client:=0

func RunAsyncTCPServer() error{
	log.Println("Starting an asynchronous TCP Server on", config.Host,config.Port)
	// since humlog linux based system use krrhe hai so 
	// so we are using epoll
	max_clients:=20000

	//create EPOLL Event Objects to hold events 
	/*
	This EPOLL event , like we ahve to register our file descriptor in EPOLL
	but when some file descriptor is ready for IO , we will get it in I/O
	This buffer will be holding file descriptor at a ready by EPOLL system call

	this events is for holding those events which are ready
	*/
	var events []syscall.EpollEvent=make([]syscall.EpollEvent,max_clients)

	//Create a socket
	/*
	here we do raw sokcet handling, we can't use raw abstaction as we are handling raw file descriptors

	so a normal socket which is an IPv4, which is NON_BLOCKING sokcet_stream

	socket_stream: i want to keep this TCP connection as soon as i got reply, basically i want to keep this connection open
	*/
	serverFD,src:=syscall.Socket(syscall.AS_INET,syscall.O_NONBLOCK | syscall.SOCK_STREAM,0)
	if err!=nil{
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

	//Bind the IP and the part
	op4:=net.ParseIP(config.Host)
	if err=syscall.Bind(serverFD,&syscall.SockaddrInet4){
		Port:config.Port,
		Addr:[4]byte{ip4[0],ip4[1],ip4[2],ip4[3]},
	}; err!=nil{
		return err
	}

	//Then we will listen, like we start to listen as FD is binded to port etc
	if err=syscall.Listen(serverFD,max_clients); err!=nil{
		return err
	}


	//Async IO Starts

	/*
	We create an EPOLL
	epoll itslef has a file descriptor of its own
	age of every fd is different inorder to identify that
	*/
	epollFD,err:=syscall.EpollCreate1(0)
	if err!=nil{
		log.Fatal(err)
	}
	defer syscall.Close(epollFD)

	/*
	you want the server (FD) to get monitored as well so we want the server to be monitored
	so we are creating an EPOLL event so isme hum kya kya montor krrhe dkeho 
	like if anytime an incoming request is coming to my server , hen basically notify me 
	so using EPOLLFD adds that to be monitored

	*/
	var socketServerEvent syscall.EpollEvent=syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:    int32(serverFD),
	}

	//listen to read evenets on the server itself 
	if err=syscall.EPOLLCtl(epollFD,syscall.EPOLL_CTL_ADD,serverFD,&socketServerEvent); err!=nil{
		return err
	}

	for{
		//basically we have to constantly monitor if an IO is ready
		//that why i am invoking EPOLL wait

		//see if any FD is ready for an IO
		nevents,e:=syscall.EpollWait(epollFD,events[:],-1)
		//EpollWait will monitor if any IO is ready, and put it in buffer(evenets buffer), if none is there then the call wouldget blocked

		if e!=nil{
			continue
		}

		for i:=0;i<nevents;i++{
			//if the socket server itself is ready for an IO
			if int(events[i].Fd)==serverFD{//each events has a File descriptors
				//if my server is ready for IO then i have to accept the incoming skcet
				//accept the incoming connection from a client
				fd,_,err:=syscall.Accept(serverFD) //you accept and get the fd between server and the new client which is getted connected
				if err!=nil{
					log.Println("err",err)
					continue
				}

				//increase the number of concurrent clients count
				con_client++;
				syscall.SetNonblock(serverFD,true)

				//add this new TCP connetion to be monitored
				var sokcetClientEvent syscall.EpollEvent=syscall.EpollEvent{
					Events:syscall.EPOLLIN
					Fd:    int32(fd),
				}
				if err:=syscall.EPOLLCtl(epollFD,syscall.EPOLL_CTL_ADD,fd,&sokcetClientEvent); err!=nil{
					log.Fatal(err)
				}
			}else{
				//if it is not my server which means that some client that is already connected to the server, then do somthing
				conn:=core.FDComm{FD:int(events[i].Fd)}
				cmd,err:=readCommand(comm)
				if err!=nil{
					syscall.Close(int(events[i].Fd))
					con_client-=1
					continue
				}
				respond(cmd,conn)
			}
		}
	}
}

/*
In synchronous we will get abstracted socket connection
while uspe hum asynchronous me fd use krte hai 

*/
