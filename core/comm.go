package core

import "syscall"

//basically we reimplemented read and write interface
type FDComm struct{
	Fd int
}

//Since we have no socket connection so i have to make system call of write over FD with the byest that i have got 
func (f FDComm) Write(b []byte)(int,error){
	return syscall.Write(f.Fd,b)
}
//everything else remains the same

func (f FDComm) Read(b []byte)(int,error){
	return syscall.Read(f.Fd,b)
}



