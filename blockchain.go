package blockchain_practice

import (
	"fmt"
	"github.com/boltdb/bolt"
	"bytes"
	"encoding/gob"
	"crypto/cipher"
	"net/mail"
	"os"
	"log"
	"errors"
	"encoding/hex"
	"crypto/ecdsa"
)

const(
	dbFile = "blockchain_%s.db"
	blocksBucket = "blocks"
	genesisCoinbaseData = "The Genesis Block"
)

type Blockchain struct{
	tail 	[]byte
	db 	*bolt.DB
}

func CreateBlockchainDB(addr, nodeID string) *Blockchain{
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if dbExist(dbFile){
		fmt.Println("dbfile exists")
		os.Exit(1)
	}
	var tail []byte
	cbtx := NewCoinbaseTX(addr, genesisCoinbaseData)
	orgBlock := NewOrgBlock(cbtx)
	db,err := bolt.Open(dbFile, 0600, nil)
	if err != nil{
		log.Panic(err)
	}
	err = db.Update(func(tx *bolt.Tx)error{
		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil{
			log.Panic(err)
		}
		err = b.Put(orgBlock.Hash, orgBlock.Serialize())
		if err != nil{
			log.Panic(err)
		}
		err = b.Put([]byte("l"), orgBlock.Hash)
		if err != nil{
			log.Panic(err)
		}
		tail = orgBlock.Hash
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	bc := &Blockchain{tail, db}
	return bc
}

func NewBlockChain(nodeID string) *Blockchain{
	dbfile := fmt.Sprintf(dbFile, nodeID)
	if dbExist(dbfile) == false{
		fmt.Println("dbfile not found, create one first")
		os.Exit(1)
	}
	var tail []byte
	db, err := bolt.Open(dbfile, 0600, nil)
	if err != nil{
		log.Panic(err)
	}
	err = db.View(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(blocksBucket))
		tail = b.Get([]byte("l"))
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	bc := &Blockchain{tail, db}
	return bc
}

func (bc *Blockchain) AddBlock(block *Block){
	err := bc.db.Update(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(blocksBucket))
		if b.Get(block.Hash) != nil{
			//fmt.Printf("%x exists\n", block.Hash)
			return nil
		}
		err := b.Put(block.Hash, block.Serialize())
		if err != nil{
			log.Panic(err)
		}
		lasthash := b.Get([]byte("l"))
		lastblockdata := b.Get(lasthash)
		lastblock := DeserializeBlock(lastblockdata)
		if lastblock.Height < block.Height{
			err := b.Put([]byte("l"), block.Hash)
			if err != nil{
				log.Panic(err)
			}
			bc.tail = block.Hash
		}
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
}

func (bc *Blockchain) FindTransaction(id []byte) (Transaction, error){
	bci := bc.Iterator()
	for {
		block := bci.Next()
		for _,tx := range block.Transactions{
			if bytes.Compare(tx.HashID, id) == 0{
				return *tx, nil
			}
		}
		if len(block.PreBlockHash) == 0{
			break
		}
	}
	return Transaction{}, errors.New("transaction not found")
}

func (bc *Blockchain) FindUTXO() map[string]TXOutputs{
	UTXO := make(map[string]TXOutputs)
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()
	for {
		block := bci.Next()
		for _,tx := range block.Transactions{
			id := hex.EncodeToString(tx.HashID)
			Outputs:
			for index,out := range tx.Vout{
				if spentTXOs[id] != nil{
					for _,preidx := range spentTXOs[id]{
						if index == preidx{
							continue Outputs
						}
					}
				}
				outs := UTXO[id]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[id] = outs
				//UTXO[id].Outputs = append(UTXO[id].Outputs, out)
			}
			if tx.IsCoinbase() == false{
				for _,in := range tx.Vin{
					txid := hex.EncodeToString(in.TxID)
					spentTXOs[txid] = append(spentTXOs[txid], in.PreOutIndex)
				}
			}
		}
		if len(block.PreBlockHash) == 0{
			break
		}
	}
	return UTXO
}

func (bc *Blockchain) Iterator() *BlockchainIterator{
	bci := &BlockchainIterator{bc.tail, bc.db}
	return bci
}

func (bc *Blockchain) GetBestHeight() int{
	var lastblock *Block
	err := bc.db.View(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(blocksBucket))
		lasthash := b.Get([]byte("l"))
		lastblockdata := b.Get(lasthash)
		//lastblockdata := b.Get(bc.tail)
		lastblock = DeserializeBlock(lastblockdata)
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	return lastblock.Height
}

func (bc *Blockchain)GetBlock(blockhash []byte) (Block, error){
	var block Block
	err := bc.db.View(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(blocksBucket))
		blockdata := b.Get(blockhash)
		if blockdata == nil{
			return errors.New("block not found")
		}
		block = *DeserializeBlock(blockdata)
		return nil
	})
	if err != nil{
		return block, err
	}
	return block, nil
}

func (bc *Blockchain) GetBlockHashes() [][]byte{
	var blockhashes [][]byte
	bci := bc.Iterator()
	for{
		block := bci.Next()
		blockhashes = append(blockhashes, block.Hash)
		if len(block.PreBlockHash) == 0{
			break
		}
	}
	return blockhashes
}

func (bc *Blockchain) MineBlock(transactions []*Transaction) *Block{
	var lasthash []byte
	var lastheight int
	for _,tx := range transactions{
		if bc.VerifyTransaction(tx) != true{
			log.Panic("invalid transaction")
		}
	}
	err := bc.db.View(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(blocksBucket))
		lasthash = b.Get([]byte("l"))
		lastblockdata := b.Get(lasthash)
		lastblock := DeserializeBlock(lastblockdata)
		lastheight = lastblock.Height
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	newblock := NewBlock(transactions, lasthash, lastheight+1)
	err = bc.db.Update(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newblock.Hash, newblock.Serialize())
		if err != nil{
			log.Panic(err)
		}
		err = b.Put([]byte("l"), newblock.Hash)
		if err != nil{
			log.Panic(err)
		}
		bc.tail = newblock.Hash
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	return newblock
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool{
	if tx.IsCoinbase(){
		return true
	}
	preTXs := make(map[string]Transaction)
	for _,in := range tx.Vin{
		pretx, err := bc.FindTransaction(in.TxID)
		if err != nil{
			log.Panic(err)
		}
		preTXs[hex.EncodeToString(pretx.HashID)] = pretx
	}
	return tx.Verify(preTXs)
}

func (bc *Blockchain) SignTransaction(tx *Transaction, private ecdsa.PrivateKey) {
	preTXs := make(map[string]Transaction)
	for _,in := range tx.Vin{
		pretx, err := bc.FindTransaction(in.TxID)
		if err != nil{
			log.Panic(err)
		}
		preTXs[hex.EncodeToString(pretx.HashID)] = pretx
	}
	tx.Sign(private, preTXs)
}

func dbExist(db string) bool{
	if _,err := os.Stat(db); os.IsNotExist(err){
		return false
	}
	return true
}