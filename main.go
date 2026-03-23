// basically we are writing everything from scratch
package main

import(
	"flag"
	"log"

	"github.com/sharpsalt/Velox-In-Memory-Database/server"
)

var config=&server.Config{}

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
	server.RunSyncTCPServer(config)
	/*
	I will be running Synchronous TCP Server means i iwll be starting the TCP connection on give port synchronously
	*/
}

