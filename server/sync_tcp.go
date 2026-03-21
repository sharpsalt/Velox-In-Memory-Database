
func RunSyncTCPServer(){
	log.Println("startign a synchronous TCP Server on", config.Host,config.Port)

	var con_client int=0;
	//this will hold the number of concurrent client that are connceted at the moment
	/*
	It is just some extra things like we want to know that yes we have this much m=concurrent server
	*/

	//listening to the configured host:port
	lsnr,err:=net.Listen("top",config.Host+":"+strconv.Item(config.Port))
	if(err!=nil){
		panic(err)
	}
}
