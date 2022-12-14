package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
)

func RSAEncrypt(secret []byte, key *rsa.PublicKey) []byte {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, key, secret, label)
	if err != nil {
		fmt.Println(err.Error())
	}

	return ciphertext
}

func RSADecrypt(secret []byte, privKey *rsa.PrivateKey) []byte {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, privKey, secret, label)
	if err != nil {
		fmt.Println(err.Error())
	}

	return plaintext
}
