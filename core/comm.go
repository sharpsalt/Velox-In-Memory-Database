package core

import (
	"bytes"
	"syscall"
	"io"
	"fmt"
)

type Client struct{
	io.ReadWriter
	Fd      int 
	enqueue RedisCmds
	isTxn   bool
}

//basically we reimplemented read and write interface
// type FDComm struct{
// 	Fd int
// }
/*this this FDComm which we have previously written is now changed to client
it has still same interface , since we don;t have to just read/write , we also do have to enqueue command , also i ahve added ki wo command transaction hai ki nahi 

so any client is connected to this then a unique object will be created for this client
*/

//Since we have no socket connection so i have to make system call of write over FD with the byest that i have got 
func (c Client) Write(b []byte)(int,error){
	return syscall.Write(c.Fd,b)
}
//everything else remains the same

func (c Client) Read(b []byte)(int,error){
	return syscall.Read(c.Fd,b)
}


func (c *Client) TxnBegin(){
	c.isTxn=true
}

func (c *Client) TxnExec() []byte{
	var out []byte
	buf:=bytes.NewBuffer(out)

	buf.WriteString(frm.Sprintf("*%d\r\n",len(c.cqueue)))

	for _,_cmd:=range c.cqueue{
		buf.Write(executeCommand(_cmd,c))
	}

	c.cqueue=make(RedisCmd,0)
	c.isTxn=false
	return buf.Bytes()
}

func (c *Client) TxnDiscard(){
	c.cqueue=make(RedisCmds,0)
	c.isTxn=false
}

func (c *Client) TxnQueue(cmd *RedisCmd){
	c.cqueue=append(c.cqueue,cmd)
}

func NewClient(fd int) *Client{
	return &Client{
		Fd:    fd,
		enqueue: amke(Rediscmds,0)
	}
}


