package core
import "errors"

/*
From a particular type encoding object if you want to get type of it
type is definately the first 4 bits 

so inorder to extract that i am right shifting by 4 and then left shift by 4
*/
func getType(te uint8) uint8{
	return (te>>4)<<4
}

func getEncoding(te uint8) uint8{
	return te & 0b00001111
}

func assertType(te uint8,t uint8)error{
	if getType(te)!=t{
		return errors.New("the operation is not permitted on this type")
	}
	return nil
}


func assertEncoding(te uint8,e uint8)error{
	if getEncoding(te)!=e{
		return errors.New("the operations is not permitted on this encoding")
	}
	return nil
}
































