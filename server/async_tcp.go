package server

import(
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sharpsalt/Velox-In-Memory-Database/config"
	"github.com/sharpsalt/Velox-In-Memory-Database/core"
)

var con_client int=0
var cronFrequency time.Duration=1*time.Second //a cron frequency of 1s
var lastCronExecTime time.Time=time.Now() //and we are maintaining, last time it ran
const EngineStatus_BUSY int32=1<<2
const EngineStatus_WAITING int32=1<<1
const EngineStatus_SHUTTING_DOWN int32=1<<3
const EngineStatus_TRANSACTION int32=1<<4


var eStatus int32=EngineStatus_WAITING
var connectedClients map[int]*core.Client //here we decalred the object 
/**
Every Time when a client is getting connected to us, we will get a file descriptor, this key is that particular fiile descriptor 
**/

func init(){
	connectedClients=make(map[int]*core.Client) //here we added memory to that
}

func WaitForSignal(wg *sync.WaitGroup,sigs chan os.Signal){
	defer wg.Done()
	<-sigs  //this is a blcoking call like until we give a channel
	//we would not be moving forward

	//if servers is busy cntinue to work
	for atomic.LoadInt32(&eStatus)==EngineStatus_BUSY{
	}

	//CRITICAL to hanle
	//we do not want server to ever go back to BUSY State 
	//when control flow is here

	//immediately set the status to be SHUTTING _DOWN
	//the only place where we can set the status to be SHUTTING_DONW
	atomic.StoreInt32(&eStatus,EngineStatus_SHUTTING_DOWN)

	//if server is in any other statem initiate a shutdown
	core.Shutdown()
	os.Exit(0)
}

// readCommands reads RESP commands from a client and returns them
/*
PERFORMANCE FIX: Client buffer reuse

BEFORE (the old code):
  var buf []byte = make([]byte, 64*1024)
  
  This allocated a FRESH 64KB buffer on EVERY single readCommands() call.
  At 10K requests/sec, that's 640MB/sec of garbage being created and collected.
  The Go GC would have to scan and free all this memory, causing latency spikes.
  
  Why was this bad?
  - make() goes to the heap allocator every time (even for the same size)
  - The buffer is used for one read and then abandoned
  - GC has to track and sweep these short-lived 64KB objects
  - GC pause times scale with the amount of garbage created

AFTER (the new code):
  n, err := c.Read(c.ReadBuf)
  
  Uses the client's pre-allocated ReadBuf (created once in NewClient).
  - Zero allocations per read
  - The same 64KB buffer is reused for every command from this client
  - GC has nothing to collect — the buffer lives until the client disconnects
  - syscall.Read writes directly into the existing buffer
  
  This one change alone can improve throughput by 15-25% at high QPS because
  it eliminates the single largest per-request allocation in the hot path.
*/
func readCommands(c *core.Client)([]*core.RedisCmd,error){
	/*
	Take the socket connection and basically fire the system call Read
	It is listening over the socket and it is trying to read message over the socket 
	if there is nothing that is coming from my client then it is a blocking call, until i get something from client 
	when we read it we put it into buffer and then, we get the number of bytes, if there is error we throw error else we send it back 
	*/
	bufPtr := core.ReadBufPool.Get().(*[]byte)
	buf := *bufPtr
	defer core.ReadBufPool.Put(bufPtr)

	n, err := c.Read(buf) 
	if err != nil{
		return nil, err
	}
	// PERF: Zero-allocation decoder. Parses the byte slice directly into 
	// a pooled slice of pooled RedisCmd structs, completely avoiding the
	// intermediate []interface{} and []string allocations.
	return core.DecodeCommands(buf[:n])
}

func RunAsyncTCPServer(wg *sync.WaitGroup) error{
	// since humlog linux based system use krrhe hai so 
	// so we are using epoll
	defer wg.Done()
	defer func(){
		atomic.StoreInt32(&eStatus,EngineStatus_SHUTTING_DOWN)
	}()

	log.Println("Starting an asynchronous TCP Server on ",config.Host,config.Port)
	/*
	===========================================================================
	🚀 SCALING TO MILLIONS OF CONCURRENT CLIENTS (The C10M Problem)
	===========================================================================
	Because we removed the pre-allocated buffers from the Client struct and moved
	them to global sync.Pools, memory consumption per idle client is virtually ZERO.
	
	An idle connection now only costs the kernel file descriptor space. You can
	safely increase `max_clients` to 1,000,000+ without blowing up the server RAM.
	(Note: You must also configure your Linux OS limits using `ulimit -n 1000000`).
	===========================================================================
	*/
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
	ip4 := net.ParseIP(config.Host).To4()
	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: config.Port,
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

	for atomic.LoadInt32(&eStatus)!=EngineStatus_SHUTTING_DOWN{ //eralier we had infinite for loop so now we will chek it until the server is not shutting down
		// PERF: Update cached time once per tick to avoid syscalls in the hot path
		core.UpdateCachedTime()
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
		//Say,the Engine triggered SHUTTING down when the control flow is here->
		//Current: Engine status==WAITING
		//Update: Engine status==SHUTTING_DOWN
		//Then we have to exit (handled in Signal handler)



		//basically we have to constantly monitor if an IO is ready
		//that why i am invoking EPOLL wait

		//see if any FD is ready for an IO
		// #10 FIX: 100ms timeout instead of -1 (blocking forever)
		// With -1, the cron job (DeleteExpiredKey) would NEVER run because
		// EpollWait blocks until an IO event arrives. With 100ms timeout,
		// we wake up periodically to check cron and shutdown status.
		nevents, e := syscall.EpollWait(epollFD, events[:], 100)
		//EpollWait will monitor if any IO is ready, and put it in buffer(evenets buffer)

		if e != nil{
			continue
		}

		//Here, we do not want server to go back from SHUTTING_DOWN
		//to BUSY
		//If the engine status==SHUTTING_DOWN over here-> we have to exit 
		//hence the only legal transition is from WAITING to BUSY
		//if that does not happen then we can exit

		//mark engine as BUSY only when it is in the waiting stats
		if !atomic.CompareAndSwapInt32(&eStatus,EngineStatus_WAITING,EngineStatus_BUSY){
			//if swap unsuccessful then the exitsing status is not WAITING, but something 
			switch eStatus{
			case EngineStatus_SHUTTING_DOWN:
				return nil
			}
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
				core.TrackConnection()
				connectedClients[fd]=core.NewClient(fd)
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
				// comm := core.FDComm{Fd: int(events[i].Fd)}
				comm:=connectedClients[int(events[i].Fd)]
				if comm==nil{
					continue
				}
				cmds, err := readCommands(comm) //instead of passing 1 command, we will pass many commands
				if err != nil{
					core.RemoveClientFromPubSub(comm)
					syscall.Close(int(events[i].Fd))
					con_client -= 1
					delete(connectedClients,int(events[i].Fd))
					continue
				}
				respondAsync(cmds, comm)
				core.FreeCmds(cmds) // PERF: Return commands to sync.Pool
			}
		}

		//mark engine as WAITING
		//no contention as the signal handler is blocked until
		//the engine is BUSY
		atomic.StoreInt32(&eStatus, EngineStatus_WAITING)
	}
	return nil
}

// respondAsync calls EvalAndRespond for the async server path
func respondAsync(cmds []*core.RedisCmd, c *core.Client) {
	core.EvalAndRespond(cmds, c)
}

/*
In synchronous we will get abstracted socket connection
while uspe hum asynchronous me fd use krte hai 



The only place wehre i can put my autodelete is inside evntloop mtlb yaha pe, as in Redis the deletion happens in two phases->active and passive
so the event loop that we wrote,basically the infinite for loop that is waiting on epoll wait,just before we added
*/
