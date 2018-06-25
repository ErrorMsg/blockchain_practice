package blockchain_practice

import (
	"log"
	"github.com/boltdb/bolt"
)

type BlockchainIterator struct{
	currentHash 	[]byte
	db 					*bolt.DB
}

func (bci *BlockchainIterator) Next() *Block{
	var block *Block
	err := bci.db.View(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(blocksBucket))
		blockdata := b.Get(bci.currentHash)
		block = DeserializeBlock(blockdata)
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	bci.currentHash = block.PreBlockHash
	return block
}
