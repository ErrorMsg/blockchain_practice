package blockchain_practice

import (
	"github.com/boltdb/bolt"
	"log"
	"encoding/hex"
)

type UTXOSet struct {
	Blockchain *Blockchain
}

func (u UTXOSet) Reindex() {
	db := u.Blockchain.db
	bucketName := []byte(utxoBucket)
	err := db.Update(func(tx *bolt.Tx)error{
		err := tx.DeleteBucket(bucketName)
		if err != nil{
			log.Panic(err)
		}
		_, err = tx.CreateBucket(bucketName)
		if err != nil{
			log.Panic(err)
		}
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	UTXO := u.Blockchain.FindUTXO()
	err = db.Update(func(tx *bolt.Tx)error{
		b := tx.Bucket(bucketName)
		for id, outs := range UTXO{
			key, err := hex.DecodeString(id)
			if err != nil{
				log.Panic(err)
			}
			err = b.Put(key, outs.Serialize())
			if err != nil{
				log.Panic(err)
			}
		}
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
}

func (u UTXOSet) FindSpendableOutputs(pubkeyhash []byte, amount int) (int, map[string][]int) {
	var unspentOutputs map[string][]int
	acc := 0
	db := u.Blockchain.db
	err := db.View(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()
		for k,v:=c.First();k!=nil;k,v=c.Next(){
			txid := hex.EncodeToString(k)
			outs := DeSerializeOutputs(v)
			for index, out := range outs.Outputs{
				if out.IsLockedWithKey(pubkeyhash) && acc < amount{
					acc += out.Value
					unspentOutputs[txid] = append(unspentOutputs[txid], index)
				}
			}
		}
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	return acc, unspentOutputs
}

func (u UTXOSet) FindUTXO(pubkeyhash []byte) []TXOutput{
	var UTXOs []TXOutput
	db := u.Blockchain.db
	err := db.View(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()
		for k,v:=c.First();k!=nil;k,v=c.Next(){
			outs := DeSerializeOutputs(v)
			for _,out := range outs.Outputs{
				if out.IsLockedWithKey(pubkeyhash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	return UTXOs
}

func (u UTXOSet) CountTransactions() int{
	count := 0
	db := u.Blockchain.db
	err := db.View(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()
		for k,_:=c.First();k!=nil;k,_=c.Next(){
			count++
		}
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
	return count
}

func (u UTXOSet) Update(block *Block) {
	db := u.Blockchain.db
	err := db.Update(func(tx *bolt.Tx)error{
		b := tx.Bucket([]byte(utxoBucket))
		for _,tx := range block.Transactions{
			if tx.IsCoinbase() == false{
				for _,in := range tx.Vin{
					updatedOuts := TXOutputs{}
					preoutsData := b.Get(in.TxID)
					outs := DeSerializeOutputs(preoutsData)
					for index, out := range outs.Outputs{
						if index != in.PreOutIndex{
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}
					if len(updatedOuts.Outputs) == 0{
						err := b.Delete(in.TxID)
						if err != nil{
							log.Panic(err)
						}
					}else{
						err := b.Put(in.TxID, updatedOuts.Serialize())
						if err != nil{
							log.Panic(err)
						}
					}
				}
			}
			newOuts := TXOutputs{}
			for _, out := range tx.Vout{
				newOuts.Outputs = append(newOuts.Outputs, out)
			}
			err := b.Put(tx.HashID, newOuts.Serialize())
			if err != nil{
				log.Panic(err)
			}
		}
		return nil
	})
	if err != nil{
		log.Panic(err)
	}
}