package blockchain_practice

import (
	"time"
	"bytes"
	"encoding/gob"
	"log"
)

type Block struct{
	Timestamp 		int64
	Transactions 	[]*Transaction
	PreBlockHash 	[]byte
	Hash 				[]byte
	Height				int
	Nonce 				int
}

func NewBlock(txs []*Transaction, preblockHash []byte, height int) *Block{
	block := &Block{time.Now().Unix(), txs, preblockHash, []byte{}, height, 0}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash[:]
	return block
}

func NewOrgBlock(coinbase *Transaction) *Block{
	return NewBlock([]*Transaction{coinbase}, []byte{}, 0)
}

func (b *Block) HashTransactions() []byte{
	var txs [][]byte
	for _,tx := range b.Transactions{
		txs = append(txs, tx.Serialize())
	}
	mTree := NewMerkleTree(txs)
	return mTree.RootNode.Data
}

func (b *Block) Serialize() []byte{
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(b)
	if err != nil{
		log.Panic(err)
	}
	return buf.Bytes()
}

func DeserializeBlock(buf []byte) *Block{
	var block *Block
	dec := gob.NewDecoder(bytes.NewReader(buf))
	err := dec.Decode(block)
	if err != nil{
		log.Panic(err)
	}
	return block
}