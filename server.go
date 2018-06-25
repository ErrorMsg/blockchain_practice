package blockchain_practice

import (
	"fmt"
	"net"
	"io"
	"bytes"
	"log"
	"encoding/gob"
	"encoding/hex"
	"io/ioutil"
)

const (
	protocol = "tcp"
	nodeVersion = 1
	commandLength = 12
)

var nodeAddress string
var miningAddress string
var knownNodes = []string{"localhost:3000"}
var blocksInTransit = []byte{}
var mempool = make(map[string]Transaction)

type addr struct{
	AddrList []string
}

type block struct{
	AddrFrom string
	Block []byte
}

type getblocks struct{
	AddrFrom string
}

type getdata struct{
	AddrFrom string
	Type string
	ID []byte
}

type inv struct{
	AddrFrom string
	Type string
	Items [][]byte
}

type tx struct{
	AddrFrom string
	Transaction []byte
}

type verzion struct{
	Version int
	BestHeight int
	AddrFrom string
}

func commandToBytes(command string) []byte{
	var data [commandLength]byte
	for i,c := range command{
		data[i] = byte(c)
	}
	return data[:]
}

func bytesToCommand(data []byte) string{
	var command []byte
	for _,b := range data{
		command = append(command, b)
	}
	return fmt.Sprintf("%s", command)
}

func extractCommand(request []byte) []byte{
	return request[:commandLength]
}

func requestBlocks() {
	for _,node := range knownNodes{
		sendGetBlocks(node)
	}
}

func sendAddr(address string){
	nodes := addr{knownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	payload := gobEncode(nodes)
	request := append(commandToBytes("addr"), payload...)
	sendData(address, request)
}

func sendBlock(address string, b *Block){
	data := block{nodeAddress, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)
	sendData(address, request)
}

func sendData(address string, data []byte){
	conn, err := net.Dial(protocol, address)
	if err != nil{
		fmt.Printf("%s unavailable", address)
		var updatedNodes []string
		for _,node := range knownNodes{
			if node != address{
				updatedNodes = append(updatedNodes, node)
			}
		}
		knownNodes = updatedNodes
		return
	}
	defer conn.Close()
	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func sendInv(address, kind string, items [][]byte){
	inventory := inv{nodeAddress, kind, items}
	payload := gobEncode(inventory)
	request := append(commandToBytes("inv"), payload...)
	sendData(address, request)
}

func sendGetBlocks(address string) {
	payload := gobEncode(getblocks{nodeAddress})
	request := append(commandToBytes("getblocks"), payload...)
	sendData(address, request)
}

func sendGetData(address, kind string, id []byte){
	payload := gobEncode(getdata{nodeAddress, kind, id})
	request := append(commandToBytes("getdata"), payload...)
	sendData(address, request)
}

func sendTx(address string, Tx *Transaction){
	data := tx{nodeAddress, Tx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)
	sendData(address, request)
}

func sendVersion(address string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()
	payload := gobencode(verzion{nodeVersion, bestHeight, nodeAddress})
	request := append(commandToBytes("version"), payload...)
	sendData(address, request)
}

func handleAddr(request []byte){
	var buf bytes.Buffer
	var payload addr
	buf.Write(request[commandLength:])
	dec := gob.NewDecoder(&buf)
	err := dec.Decode(&payload)
	if err != nil{
		log.Panic(err)
	}
	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Printf("%d known nodes\n", len(knownNodes))
	requestBlocks()
}

func handleBlock(request []byte, bc *Blockchain){
	var buf bytes.Buffer
	var payload block
	buf.Write(request[commandLength:])
	dec := gob.NewDecoder(&buf)
	err := dec.Decode(&payload)
	if err != nil{
		log.Panic(err)
	}
	blockData := payload.Block
	block := DeserializeBlock(blockData)
	fmt.Println("received new block")
	bc.AddBlock(block)
	fmt.Printf("block %x added\n", block.Hash)
	if len(blocksInTransit) > 0{
		blockHash := blocksInTransit[0]
		sendGetData(payload.AddrFrom, "block", blockHash)
		blocksInTransit = blocksInTransit[1:]
	}else{
		UTXOSet := UTXOSet{bc}
		UTXOSet.Reindex()
	}
}

func handleInv(request []byte, bc *Blockchain){
	var buf bytes.Buffer
	var payload inv
	buf.Write(request[commandLength:])
	dec := gob.NewDecoder(&buf)
	err := dec.Decode(&payload)
	if err != nil{
		log.Panic(err)
	}
	fmt.Printf("received inventory with %d %s\n", len(payload.Items), payload.Type)
	if payload.Type == "block"{
		blocksInTransit = payload.Items
		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom, "block", blockHash)
		newInTransit := [][]byte{}
		for _,b := range blocksInTransit{
			if bytes.Compare(b, blockHash) != 0{
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}
	if payload.Type == "tx"{
		txid := payload.Items[0]
		if mempool[hex.EncodeToString(txid)].HashID == nil{
			sendGetData(payload.AddrFrom, "tx", txid)
		}
	}
}

func handleGetBlocks(request []byte, bc *Blockchain){
	var buf bytes.Buffer
	var payload getblocks
	buf.Write(request[commandLength:])
	dec := gob.NewDecoder(&buf)
	err := dec.Decode(&payload)
	if err != nil{
		log.Panic(err)
	}
	blocks := bc.GetBlockHashes()
	sendInv(payload.AddrFrom, "block", blocks)
}

func handleGetData(request []byte, bc *Blockchain){
	var buf bytes.Buffer
	var payload getdata
	buf.Write(request[commandLength:])
	dec := gob.NewDecoder(&buf)
	err := dec.Decode(&payload)
	if err != nil{
		log.Panic(err)
	}
	if payload.Type == "block"{
		block, err := bc.GetBlock([]byte(payload.ID))
		if err != nil{
			log.Panic(err)
		}
		sendBlock(payload.AddrFrom, &block)
	}
	if payload.Type == "tx"{
		txid := hex.EncodeToString(payload.ID)
		tx := mempool[txid]
		sendTx(payload.AddrFrom, &tx)
	}
}

func handleTx(request []byte, bc *Blockchain){
	var buf bytes.Buffer
	var payload tx
	buf.Write(request[commandLength:])
	dec := gob.NewDecoder(&buf)
	err := dec.Decode(&payload)
	if err != nil{
		log.Panic(err)
	}
	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	mempool[hex.EncodeToString(tx.HashID)] = tx
	if nodeAddress == knownNodes[0]{
		for _,node := range knownNodes{
			if node != nodeAddress && node != payload.AddrFrom{
				sendInv(node, "tx", [][]byte{tx.HashID})
			}
		}
	}else{
		if len(mempool) >= 2 && len(miningAddress) > 0{
			MineTransactions:
				var txs []*Transaction
				for id := range mempool{
					tx := mempool[id]
					if bc.VerifyTransaction(&tx){
						txs = append(txs, &tx)
					}
				}
				if len(txs) == 0{
					fmt.Println("all transactions invald")
					return
				}
				cbtx := NewCoinbase(miningAddress, "")
				txs = append(txs, cbtx)
				newBlock := bc.MineBlock(txs)
				UTXOSet := UTXOSet{bc}
				UTXOSet.Reindex()
				fmt.Println("new block mined")
				for _,tx := range txs {
					id := hex.EncodeToString(tx.HashID)
					delete(mempool, id)
				}
				for _,node := range knownNodes{
					if node != nodeAddress{
						sendInv(node, "block", [][]byte{newBlock.Hash})
					}
				}
				if len(mempool) > 0{
					goto MineTransactions
				}
		}
	}
}

func handleVersion(request []byte, bc *Blockchain){
	var buf bytes.Buffer
	var payload verzion
	buf.Write(request[commandLength:])
	dec := gob.NewDecoder(&buf)
	err := dec.Decode(&payload)
	if err != nil{
		log.Panic(err)
	}
	myBestHeight := bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight
	if myBestHeight < foreignerBestHeight{
		sendGetBlocks(payload.AddrFrom)
	}else if myBestHeight > foreignerBestHeight{
		sendVersion(payload.AddrFrom,bc)
	}
	if !nodeIsKnown(payload.AddrFrom){
		knownNodes = append(knownNodes, payload.AddrFrom)
	}
}

func handleConnection(conn net.Conn, bc *Blockchain){
	request, err := ioutil.ReadAll(conn)
	if err != nil{
		log.Panic(err)
	}
	command := bytesToCommand(request[:commandLength])
	fmt.Printf("received %s command\n", command)
	switch command{
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		fmt.Println("unknown command")
	}
	conn.Close()
}

func StartServer(nodeID, minerAddress string){
	nodeAddress = fmt.Sprint("localhost:%s", nodeID)
	miningAddress = minerAddress
	ln,err := net.Listen(protocol, nodeAddress)
	if err != nil{
		log.Panic(err)
	}
	defer ln.Close()
	bc := NewBlockChain(nodeID)
	if nodeAddress != knownNodes[0]{
		sendVersion(knownNodes[0], bc)
	}
	for {
		conn, err := ln.Accept()
		if err != nil{
			log.Panic(err)
		}
		go handleConnection(conn, bc)
	}
}

func gobEncode(data interface{}) []byte{
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil{
		log.Panic(err)
	}
	return buf.Bytes()
}

func nodeIsKnown(addr string) bool{
	for _,node := range knownNodes{
		if node == addr{
			return true
		}
	}
	return false
}