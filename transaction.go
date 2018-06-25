package blockchain_practice

import (
	"fmt"
	"encoding/gob"
	"crypto/sha256"
	"log"
	"bytes"
	"encoding/hex"
	"crypto/ecdsa"
	"crypto/rand"
	"strings"
	"crypto/elliptic"
	"math/big"
)

type Transaction struct{
	HashID 		[]byte
	Vin		 		[]TXInput
	Vout 			[]TXOutput
}

type TXOutput struct{
	Value 				int
	PubKeyHash 	[]byte
}

type TXOutputs struct{
	Outputs []TXOutput
}

type TXInput struct{
	TxID 				[]byte
	PreOutIndex	int
	Signature 		[]byte
	PubKey 			[]byte
}

func (tx Transaction) IsCoinbase() bool{
	return len(tx.Vin) == 1 && len(tx.Vin[0].TxID) == 0 && tx.Vin[0].PreOutIndex == -1
}

func (tx Transaction) Serialize() []byte{
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx)
	if err != nil{
		log.Panic(err)
	}
	return buf.Bytes()
}

func (tx *Transaction) Hash() []byte{
	var hash [32]byte
	txcopy := *tx
	txcopy.HashID = []byte{}
	hash = sha256.Sum256(txcopy.Serialize())
	return hash[:]
}

func (tx *Transaction) Sign(private ecdsa.PrivateKey, preTXs map[string]Transaction){
	if tx.IsCoinbase(){
		return
	}
	for _,in := range tx.Vin{
		if preTXs[hex.EncodeToString(in.TxID)]  == nil{
			log.Panic("previous transaction not correct")
		}
	}
	txcopy := tx.TrimmedCopy()
	for inidx,in := range txcopy.Vin{
		pretx := preTXs[hex.EncodeToString(in.TxID)]
		txcopy.Vin[inidx].Signature = nil
		txcopy.Vin[inidx].PubKey = pretx.Vout[in.PreOutIndex].PubKeyHash
		dataToSign := fmt.Sprintf("%x\n", txcopy)
		r,s,err := ecdsa.Sign(rand.Reader, &private, []byte{dataToSign})
		if err != nil{
			log.Panic(err)
		}
		/*txcopy.HashID = Hash(txcopy)
		txcopy.Vin[inidx].PubKey = nil
		r,s,err := ecdsa.Sign(rand.Reader, &private, txcopy.HashID)
		if err != nil{
			log.Panic(err)
		}*/
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Vin[inidx].Signature = signature
		txcopy.Vin[inidx].PubKey = nil
	}
}

func (tx *Transaction) String() string{
	var lines []string
	lines = append(lines, fmt.Sprintf("Transaction: %x: ", tx.HashID))
	for i, in := range tx.Vin{
		lines = append(lines, fmt.Sprintf("---Input is %d:", i))
		lines = append(lines, fmt.Sprintf("---TxID is %x:", in.TxID))
		lines = append(lines, fmt.Sprintf("---PreOutIndex is %d:", in.PreOutIndex))
		lines = append(lines, fmt.Sprintf("---Signature is %x:", in.Signature))
		lines = append(lines, fmt.Sprintf("---PubKey is %x:", in.PubKey))
	}
	for i,out := range tx.Vout{
		lines = append(lines, fmt.Sprintf("---Output is %d:", i))
		lines = append(lines, fmt.Sprintf("---Value is %d:", out.Value))
		lines = append(lines, fmt.Sprintf("---PubKeyHash is %x:", out.PubKeyHash))
	}
	return strings.Join(lines, "\n")
}

func (tx *Transaction) TrimmedCopy() Transaction{
	var inputs []TXInput
	var outputs []TXOutput
	for _,in := range tx.Vin{
		input := TXInput{in.TxID, in.PreOutIndex, nil, nil}
		inputs = append(inputs, input)
	}
	for _,out := range tx.Vout{
		output := TXOutput{out.Value, out.PubKeyHash}
		outputs = append(outputs, output)
	}
	txcopy := Transaction{tx.HashID, inputs, outputs}
	return txcopy
}

func (tx *Transaction) Verify(preTXs map[string]Transaction) bool{
	if tx.IsCoinbase(){
		return true
	}
	for _,in := range tx.Vin{
		if preTXs[hex.EncodeToString(in.TxID)].HashID == nil{
			log.Panic("preivous transaction not correct")
		}
	}
	txcopy := tx.TrimmedCopy()
	curve := elliptic.P256()
	for inidx, in := range tx.Vin{
		pretx := preTXs[hex.EncodeToString(in.TxID)]
		txcopy.Vin[inidx].Signature = nil
		txcopy.Vin[inidx].PubKey = pretx.Vout[in.PreOutIndex].PubKeyHash

		r := big.Int{}
		s := big.Int{}
		siglen := len(in.Signature)
		r.SetBytes(in.Signature[:(siglen/2)])
		s.SetBytes(in.Signature[(siglen/2):])

		x := big.Int{}
		y := big.Int{}
		keylen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(keylen/2)])
		y.SetBytes(in.PubKey[(keylen/2):])

		dataToVerify := fmt.Sprintf("%x\n", txcopy)
		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false{
			return false
		}
		txcopy.Vin[inidx].PubKey = nil
	}
	return true
}

func NewCoinbaseTX(to, data string) *Transaction{
	if data == ""{
		randData := make([]byte,20)
		_,err := rand.Read(randData)
		if err != nil{
			log.Panic(err)
		}
		data = fmt.Sprintf("%x", randData)
		//data = fmt.Sprintf("reward to %s\n", to)
	}
	txin := TXInput{[]byte{}, -1,  nil,[]byte(data)}
	txout := NewTXOutput{subsidy, to}
	tx := Transaction{[]byte{}, []TXInput{txin}, []TXOutput{txout}}
	tx.HashID = tx.Hash()
	return &tx
}

func NewUTXOTransaction(wallet *Wallet, to string, amount int, UTXOSet *UTXOSet) *Transaction{
	var inputs []TXInput
	var outputs []TXOutput
	pubkeyhash := HashPubKey(wallet.PublicKey)
	acc, validOutputs := UTXOSet.FindSpendableOutputs{pubkeyhash, amount}
	if acc < amount{
		log.Panic("no enough funds")
	}
	for id, outidxs := range validOutputs{
		txid,err := hex.DecodeString(id)
		if err != nil{
			log.Panic(err)
		}
		for _,idx := range outidxs{
			input := TXInput{txid, idx, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}
	from := fmt.Sprintf("%s", wallet.GetAddress())
	outputs = append(outputs, *NewTXOutput(to, amount))
	if acc > amount{
		outputs = append(outputs, *NewTXOutput(from, acc - amount))
	}
	tx := Transaction{[]byte{}, inputs, outputs}
	tx.HashID = tx.Hash()
	UTXOSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey)
	return &tx
}

func DeserializeTransaction(data []byte) Transaction{
	var transaction Transaction
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&transaction)
	if err != nil{
		log.Panic(err)
	}
	return transaction
}

func (in *TXInput) UseKey (pubkeyhash []byte) bool{
	lockkey := HashPubKey(in.PubKey)
	return bytes.Compare(lockKey, pubkeyhash) == 0
}

func (out *TXOutput) Lock(address []byte){
	pubkeyhash := Base58Decode(address)
	pubkeyhash = pubkeyhash[1:len(pubkeyhash)-4]
	out.PubKeyHash = pubkeyhash
}

func (out *TXOutput) IsLockedWithKey(pubkeyhash []byte) bool{
	return bytes.Compare(out.PubKeyHash, pubkeyhash) == 0
}

func NewTXOutput(address string, value int) *TXOutput{
	out := &TXOutput{value, nil}
	out.Lock([]byte(address))
	return out
}

func (outs TXOutputs) Serialize() []byte{
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(outs)
	if err != nil{
		log.Panic(err)
	}
	return buf.Bytes()
}

func DeSerializeOutputs(data []byte) TXOutputs{
	var outs TXOutputs
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outs)
	if err != nil{
		log.Panic(err)
	}
	return outs
}