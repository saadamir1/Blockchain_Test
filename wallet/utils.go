package wallet

import (
	"bytes"
	"log"

	"github.com/mr-tron/base58"
)

func Base58Encode(input []byte) []byte {
	encode := base58.Encode(input)
	return []byte(encode)
}

func Base58Decode(input []byte) []byte {
	decode, err := base58.Decode(string(input))
	if err != nil {
		log.Panic(err)
	}
	return decode
}

// ValidateAddress checks if an address is valid
func ValidateAddress(address string) bool {
	if len(address) < checksumLength {
		return false
	}
	
	fullHash := Base58Decode([]byte(address))
	actualChecksum := fullHash[len(fullHash)-checksumLength:]
	version := fullHash[0]
	pubKeyHash := fullHash[1 : len(fullHash)-checksumLength]
	targetChecksum := CheckSum(append([]byte{version}, pubKeyHash...))

	return bytes.Equal(actualChecksum, targetChecksum)
}