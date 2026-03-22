/*
Basically encoding and decoding of values will go here

*/

//read a RESP encoded simple string from the data and returns 
//the string, the data, and the error 
func readSimpleString(data []byte)(string,int,error){
	//first character +b 
	pos:=1
	for ; data[pos]!='\r';pos++{
	}

	return string(data[1:pos]),pos+2,nil
}

//read a RESP encoded error from data and returns 
//the error string, the delta, and the error
//It is almsot same as ReadSimpleString, only the difference is it starts with - instead of +
func readError(data []byte)(string,int,error){
	return readSimpleString(data)
}

func DecodeOne(data []byte)(interface{},int,error){
	if len(data)==0{
		return nil,0,error.New("no data");
	}
	switch data[0]{
	case '+':
		return readSimpleString(data)
	case '-':
		return readError(data)
	case ':':
		return readInt64(data)
	case '$':
		return readBulkString(data)
	case '*':
		return readArray(data)
	}
	return nil,0,nil
}


func Decode(data []byte)(interface{},error){
	/*
	Decode function will take a slice of byte called as Data
	and it returns the actuals object, and also return an optional error

	So we may get extremely large slice of byte but the decodeOne willl decode the first RESP value of it 
	DecodeOne will decode each of it and return value,int,and optional error.
	*/
	if len(data)==0{
		return nil,errors.New("no data")
	}

	value,_,err:=DecodeOne(data)
	return value,err
}

