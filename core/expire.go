//Delete all the expired keys- the active way
//Sampling approach: https://redis.io/commands/expire/
func DeleteExpiredKey(){
	for{
		frac:=expireSample()
		//if the sample had less than 25% keys required
		//we break the loop
		if frac<0.25{
			break
		}
	}
	log.Println("deleted the expired but undeleted logs, total keys ",len(store));
}