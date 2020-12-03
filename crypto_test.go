///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"crypto/rand"
	"testing"
)

// TestCrypto smoke tests the crypto helper functions
func TestCrypto(t *testing.T) {
	plaintext := []byte("Hello, World!")
	password := "test_password"
	ciphertext := encrypt(plaintext, password, rand.Reader)
	decrypted, err := decrypt(ciphertext, password)
	if err != nil {
		t.Errorf("%+v", err)
	}

	for i := 0; i < len(plaintext); i++ {
		if plaintext[i] != decrypted[i] {
			t.Errorf("%b != %b", plaintext[i], decrypted[i])
		}
	}
}
