package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"

	"github.com/GermanVor/devops-pet-project/internal/common"
)

func RSAEncrypt(secret []byte, key *rsa.PublicKey) ([]byte, error) {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader

	hash := sha256.New()
	limit := key.Size() - 2*hash.Size() - 2
	var res []byte

	for _, chunk := range common.Chunks(secret, limit) {
		encryptedChunk, err := rsa.EncryptOAEP(hash, rng, key, chunk, label)
		if err != nil {
			return nil, err
		}

		res = append(res, encryptedChunk[:]...)
	}

	return res, nil
}

func RSADecrypt(secret []byte, privKey *rsa.PrivateKey) ([]byte, error) {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	hash := sha256.New()

	var res []byte

	for _, chunk := range common.Chunks(secret, 512) {
		decryptedChunk, err := rsa.DecryptOAEP(hash, rng, privKey, chunk, label)
		if err != nil {
			return nil, err
		}

		res = append(res, decryptedChunk[:]...)
	}

	return res, nil
}
