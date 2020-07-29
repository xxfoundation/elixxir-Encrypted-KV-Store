////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Elixxir                                                    /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package ekv

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"io"
)

func hashPassword(password string) []byte {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	return hasher.Sum(nil)
}

func initAESGCM(password string) cipher.AEAD {
	aesCipher, _ := aes.NewCipher(hashPassword(password))
	// NOTE: We use gcm as it's authenticated and simplest to set up
	aesGCM, err := cipher.NewGCM(aesCipher)
	if err != nil {
		panic(fmt.Sprintf("Could not init AES GCM mode: %s",
			err.Error()))
	}
	return aesGCM
}

func encrypt(data []byte, password string) []byte {
	aesGCM := initAESGCM(password)
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(fmt.Sprintf("Could not generate nonce: %s", err.Error()))
	}
	ciphertext := aesGCM.Seal(nonce, nonce, data, nil)
	return ciphertext
}

// Use the prefix hash of the data as the nonce, so you can always generate
// the same encryption for the data (this is used to generate keys)
func encryptHashNonce(data []byte, password string) []byte {
	aesGCM := initAESGCM(password)
	h, _ := blake2b.New256(nil)
	h.Write(data)
	nonce := h.Sum(nil)[:aesGCM.NonceSize()]
	ciphertext := aesGCM.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func decrypt(data []byte, password string) ([]byte, error) {
	aesGCM := initAESGCM(password)
	nonceLen := aesGCM.NonceSize()
	nonce, ciphertext := data[:nonceLen], data[nonceLen:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot decrypt with password!")
	}
	return plaintext, nil
}
