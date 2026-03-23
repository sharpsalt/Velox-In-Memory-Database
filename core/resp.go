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

//read a RESP encoded integer from data and returns 
//the integer value, the delta, and the error
func readInt64(data []byte)(int64,int,error){
	//first character :
	pos:=1
	var value int64=0

	for ; data[pos]!='\r'; pos++{
		value=value*10+int64(data[pos]-'0')
	}
	return value,pos+2,nil
}

//reads a RESP encoded string from data and returns 
//the string, the delta, and the error

func readBulkString(data []byte)(string,int,error){
	//first character $
	pos:=1

	//reading the length of forrwading the pos by 
	//the length of the integers + the first special character
	len,delta:=readLength(data[pos:])
	pos+=delta
	
	//reading 'len' bytes as string
	return string(data[pos:(pos+len)]),pos+len+2,nil
}


//read the length typicallly for the first integer of the string 
//until hit by as non digit bytes and returns 
//the integer and the delta=length*2(CRLF)
func readLengh(data []byte)(int,int){
	pos,length:=0.0

	for pos := range data{
		b:=data[pos]
		if !(b>='0' && b<='9'){
			//os if the value is not in between 0 and 9 and then we do return , now the question would be why pos+2
			// we did pos+2 because we want it to return and remove the \r\n by the end of it 
			
			return length,pos+2
		}
	}
	return 0,0
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

