package blockchain_practice

import (
	"bytes"
	"encoding/binary"
	"log"
)

func IntToHex(nums int64) []byte{
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, nums)
	if err != nil{
		log.Panic(err)
	}
	return buf.Bytes()
}

func ReverseBytes(data []byte){
	for i,j:=0,len(data)-1;i<j;i,j=i+1,j-1{
		data[i], data[j] = data[j], data[i]
	}
}