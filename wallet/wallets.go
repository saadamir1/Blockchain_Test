package wallet

import (
	"bytes"
	"crypto/x509"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const walletFile = "./tmp/wallets.data"

type SerializableWallet struct {
	PrivateKey []byte
	PublicKey  []byte
}

type Wallets struct {
	Wallets map[string]*SerializableWallet
}

func CreateWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*SerializableWallet)
	err := wallets.LoadFile()
	return &wallets, err
}

func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	
	privateKeyBytes, err := x509.MarshalECPrivateKey(&wallet.PrivateKey)
	if err != nil {
		log.Panic(err)
	}
	
	address := fmt.Sprintf("%s", wallet.Address())
	ws.Wallets[address] = &SerializableWallet{
		PrivateKey: privateKeyBytes,
		PublicKey:  wallet.PublicKey,
	}
	
	ws.SaveFile()
	return address
}

func (ws *Wallets) GetWallet(address string) *Wallet {
	serializable := ws.Wallets[address]
	if serializable == nil {
		return nil
	}
	
	privateKey, err := x509.ParseECPrivateKey(serializable.PrivateKey)
	if err != nil {
		log.Panic(err)
	}
	
	return &Wallet{
		PrivateKey: *privateKey,
		PublicKey:  serializable.PublicKey,
	}
}

func (ws *Wallets) LoadFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		return err
	}

	var wallets Wallets
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		return err
	}

	ws.Wallets = wallets.Wallets
	return nil
}

func (ws *Wallets) SaveFile() {
	var content bytes.Buffer
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}
func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}
	return addresses
}