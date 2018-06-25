package blockchain_practice

import(
	"bytes"
	"math/big"
	"math"
	"fmt"
	"crypto/sha256"
	"encoding/binary"
	"log"
)

const targetbits = 16
var maxNonce = math.MaxInt64

type ProofOfWork struct{
	block 	*Block
	target 	*big.Int
}

func NewProofOfWork(block *Block) *ProofOfWork{
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetbits))
	pow := &ProofOfWork{block, target}
	return pow
}

func (pow *ProofOfWork) prepareData(nonce int) []byte{
	data := bytes.Join{
		[][]byte{
			pow.block.PreBlockHash,
			pow.block.HashTransactions(),
			IntToHex(pow.block.Timestamp),
			IntToHex(int64(targetbits)),
			IntToHex(int64(nonce)),
		},
		[]byte{},
	}
	return data
}

func (pow *ProofOfWork) Run() (int, []byte){
	var hash [32]byte
	var hashInt big.Int
	nonce := 0
	fmt.Println("mining new block")
	for nonce < maxNonce{
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		if math.Remainder(float64(nonce), 100000) == 0{
			fmt.Printf("\r%x", hash)
		}
		hashInt.SetBytes(hash[:])
		if hashInt.Cmp(pow.target) == -1{
			break
		}else{
			nonce++
		}
	}
	fmt.Println()
	return nonce, hash[:]
}

func (pow *ProofOfWork) Validate() bool{
	var hashInt big.Int
	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])
	isValid := hashInt.Cmp(pow.target) == -1
	return isValid
}