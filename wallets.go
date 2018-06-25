package blockchain_practice

import (
	"fmt"
	"os"
	"io/ioutil"
	"log"
	"encoding/gob"
	"crypto/elliptic"
	"bytes"
)

type Wallets struct{
	Wallets map[string]*Wallet
}

func NewWallets(nodeID string) (*Wallets, error){
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)
	err := wallets.LoadFromFile(nodeID)
	return &wallets, err
}

func (ws *Wallets) CreateWallet() string{
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())
	ws.Wallets[address] = wallet
	return address
}

func (ws *Wallets) GetAddresses() []string{
	var addrs []string
	for addr := range ws.Wallets{
		addrs = append(addrs, addr)
	}
	return addrs
}

func (ws *Wallets) GetWallet(address string) Wallet{
	return *ws.Wallets[address]
}

func (ws *Wallets) LoadFromFile(nodeID string) error{
	walletFile := fmt.Sprintf(walletFile, nodeID)
	if _,err := os.Stat(walletFile); os.IsNotExist(err){
		return err
	}
	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil{
		log.Panic(err)
	}
	var wallets Wallets
	gob.Register(elliptic.P256())
	dec := gob.NewDecoder(bytes.NewReader(fileContent))
	err := dec.Decode(&wallets)
	if err != nil{
		log.Panic(err)
	}
	ws.Wallets = wallets.Wallets
	return nil
}

func (ws *Wallets) SaveToFile(nodeID string){
	var content bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeID)
	gob.Register(elliptic.P256())
	enc := gob.NewEncoder(&content)
	err := enc.Encode(ws)
	if err != nil{
		log.Panic(err)
	}
	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil{
		log.Panic(err)
	}
}