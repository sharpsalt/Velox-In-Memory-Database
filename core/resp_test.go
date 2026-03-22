package core_test

import{
	"fmt",
	"testing",
}

func TestSimpleStringDecode(t *testing.T){
	cases:=map[string]sting{
		"+OK\r\n":"OK",
	}
	for k,v := range cases{
		value,_:=core.Decode([]byte(k))
		if v!=value{
			t.Fail()
		}
	}
}


func TestError(t *testing.T){
	cases:=map[string]string{
		"-Error Message\r\n":"Error message"
	}
	for k,v := range cases{
		value,_:=core.Decode([]byte(k))
		if v!=value{
			t.Fail()
		}
	}
}
