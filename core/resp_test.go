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

func TestInt64(t *testing.T){
	cases:=map[string]string{
		":0\r\n":0,
		":1000\r\n":1000,
	}

	for k,v := range cases{
		value,_:=core.Decode([]byte(k))
		if v!=value{
			t.Fail()
		}
	}
}

func TestCulkStringDecode(t *Testing.T){
	cases:=map[string]string{
		"$5\r\nhello\r\n":"hello",
		"$0\r\n\r\n":"",
	}
	for k,v := range cases{
		value,_:=core.Decode([]byte(k))
		if v!=value{
			t.Fail()
		}
	}
}
