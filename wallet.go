package blockchain_practice

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"log"
	"golang.org/x/crypto/ripemd160"
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
)

const version = byte(0x00)
const addressChecksumLen = 4

type Wallet struct{
	PrivateKey ecdsa.PrivateKey
	PublicKey []byte
}

func NewWallet() *Wallet{
	private, public := newKeyPair()
	wallet := &Wallet{private, public}
	return wallet
}

func (w Wallet) GetAddress() []byte{
	pubkeyhash := HashPubKey(w.PublicKey)
	versionedPayload := append([]byte{version}, pubkeyhash...)
	checksum := checkSum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)
	return address
}

func HashPubKey(pubkey []byte) []byte{
	pub256 := sha256.Sum256(pubkey)
	RIPEMD160Hasher := ripemd160.New()
	_,err := RIPEMD160Hasher.Write(pub256[:])
	if err != nil{
		log.Panic(err)
	}
	pub160 := RIPEMD160Hasher.Sum(nil)
	return pub160
}

func ValidateAddress(address string) bool{
	pubkeyhash := Base58Decode([]byte(address))
	length := len(pubkeyhash) - addressChecksumLen
	addrcs := pubkeyhash[length:]
	version := pubkeyhash[0]
	pubkeyhash = pubkeyhash[1:length]
	targetcs := checkSum(append([]byte{version}, pubkeyhash...))
	return bytes.Compare(addrcs, targetcs) == 0
}

func checkSum(payload []byte) []byte{
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	return secondSHA[:addressChecksumLen]
}

func newKeyPair() (ecdsa.PrivateKey, []byte){
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil{
		log.Panic(err)
	}
	public := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, public
}

