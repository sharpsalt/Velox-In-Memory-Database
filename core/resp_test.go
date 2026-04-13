package core_test

import (
	"fmt"
	"testing"

	"github.com/sharpsalt/Velox-In-Memory-Database/core"
)

func TestSimpleStringDecode(t *testing.T){
	cases := map[string]string{
		"+OK\r\n": "OK",
	}
	for k, v := range cases{
		values, _ := core.Decode([]byte(k))
		// Decode returns []interface{}, get the first value
		if len(values) == 0 {
			t.Fail()
			continue
		}
		value := values[0].(string)
		if v != value{
			t.Fail()
		}
	}
}

func TestError(t *testing.T){
	cases := map[string]string{
		"-Error Message\r\n": "Error Message",
	}
	for k, v := range cases{
		values, _ := core.Decode([]byte(k))
		// Decode returns []interface{}, get the first value
		if len(values) == 0 {
			t.Fail()
			continue
		}
		value := values[0].(string)
		if v != value{
			t.Fail()
		}
	}
}

func TestInt64(t *testing.T) {
	cases := map[string]int64{
		":0\r\n":    0,
		":1000\r\n": 1000,
	}

	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		// Decode returns []interface{}, get the first value
		if len(values) == 0 {
			t.Fail()
			continue
		}
		value := values[0].(int64)
		if v != value {
			t.Fail()
		}
	}
}

func TestBulkStringDecode(t *testing.T) {
	cases := map[string]string{
		"$5\r\nhello\r\n": "hello",
		"$0\r\n\r\n":      "",
	}
	for k, v := range cases{
		values, _ := core.Decode([]byte(k))
		// Decode returns []interface{}, get the first value
		if len(values) == 0 {
			t.Fail()
			continue
		}
		value := values[0].(string)
		if v != value{
			t.Fail()
		}
	}
}

func TestArrayDecode(t *testing.T) {
	cases := map[string][]interface{}{
		"*0\r\n":                                          {},
		"*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n":            {"hello", "world"},
		"*3\r\n:1\r\n:2\r\n:3\r\n":                       {int64(1), int64(2), int64(3)},
		"*5\r\n:1\r\n:2\r\n:3\r\n:4\r\n$5\r\nHello\r\n":       {int64(1), int64(2), int64(3), int64(4), "Hello"},
		"*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n": {[]interface{}{int64(1), int64(2), int64(3)}, []interface{}{"Hello", "World"}},
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		// Decode returns []interface{}, get the first element
		if len(values) == 0 {
			t.Fail()
			continue
		}
		value := values[0]
		
		switch array := value.(type) {
		case []interface{}:
			if len(array) != len(v) {
				t.Fail()
			}
			for i := range array {
				if fmt.Sprintf("%v", v[i]) != fmt.Sprintf("%v", array[i]) {
					t.Fail()
				}
			}
		default:
			t.Fail()
		}
	}
}

