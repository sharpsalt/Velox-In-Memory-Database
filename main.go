// basically we are writing everything from scratch
package main 

import(
"flag"
"log"
"os"
"os/signal"
"syscall"
"sync"  
"github.com/sharpsalt/Velox-In-Memory-Database/server"
"github.com/sharpsalt/Velox-In-Memory-Database/config"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"sync"  
	"github.com/sharpsalt/Velox-In-Memory-Database/server"
	"github.com/sharpsalt/Velox-In-Memory-Database/config"
	"github.com/sharpsalt/Velox-In-Memory-Database/core"
)

func setupFlag(){
	/*
	we are setting 2 flags, these are the command lines we will give when we do execute something
	when we start database we will specify at which database our server is listening to
	Basically redis is 7379 and other db as 0.0.0.0 as host
	from whcih ip we should be accepting incoming the connection
	*/
	flag.StringVar(&config.Host, "host", "0.0.0.0", "hots for the server")
	flag.IntVar(&config.Port, "port", 7379, "port for the server")
	flag.Parse()
}

func main(){
	setupFlag() //we will setup the flags firt
	log.Println("hello!! is it really running")

	// #12: Replay AOF file on startup to restore persisted data
	core.InitStore()

	// err:=server.RunAsyncTCPServer(config)
	// if err!=nil{
	// 	log.Println("Error starting server:", err)
	// 	return
	// }
	/*
	I will be running Synchronous TCP Server means i iwll be starting the TCP connection on give port synchronously
	*/
	var sigs cha os.Signal=make(chan os.Signal,1)
	var sigs chan os.Signal=make(chan os.Signal,1)
	/*
	we will listen to interrupts or signals is through a channel
	so here we are creating a channel who accept of type os.Signal

	and then we are registering it to listen SIGTERM or SIGINT
	*/
	signal.Notify(sigs,syscall.SIGTERM,syscall.SIGINT)
	var wg sync.WaitGroup
	wg.Add(2)

	go server.RunAsyncTCPServer(&wg)
	go server.WaitForSignal(&wg,sigs)
	/*
	We are spinning up 2 goroutine

	earlier we had only 1 thread, so currently we have 2thread, 1 is for actual signal(vo mera asynctcpserver)
	and we will wait till completion 
	*/
	wg.Wait()
}
