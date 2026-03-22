/*
Basically encoding and decoding of values will go here

*/

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

