///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/ekv/portableOS"
	"io"
	"os"
	"sync"
)

// Filestore implements an ekv by reading and writing to files in a
// directory.
type Filestore struct {
	basedir  string
	password string
	sync.RWMutex
	keyLocks map[string]*sync.RWMutex
	csprng   io.Reader
}

// NewFilestore returns an initialized filestore object or an error
// if it can't read and write to the directory/.ekv.1/2 file. Note that
// this file is not used other than to verify read/write capabilities on the
// directory.
func NewFilestore(basedir, password string) (*Filestore, error) {
	return NewFilestoreWithNonceGenerator(basedir, password, rand.Reader)
}

// NewFilestoreWithNonceGenerator returns an initialized filestore object that
// uses a custom RNG for Nonce generation.
func NewFilestoreWithNonceGenerator(basedir, password string,
	csprng io.Reader) (*Filestore, error) {
	// Create the directory if it doesn't exist, otherwise do nothing.
	err := portableOS.MkdirAll(basedir, 0700)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Get the path to the "ekv" file
	ekvPath := basedir + string(os.PathSeparator) + ".ekv"
	expectedContents := []byte("version:1")

	// Try to read the .ekv.1/2 file, if it exists then we check
	// it's contents
	ekvCiphertext, err := read(ekvPath)
	if !os.IsNotExist(err) {
		if err != nil {
			return nil, errors.WithStack(err)
		} else if ekvCiphertext != nil {
			ekvContents, err := decrypt(ekvCiphertext, password)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			if !bytes.Equal(ekvContents, expectedContents) {
				return nil, errors.Errorf("Bad decryption: "+
					"%s != %s", ekvContents,
					expectedContents)
			}
		}
	}

	// Now try to write the .ekv file which also reads and verifies what
	// we write
	err = write(ekvPath, encrypt(expectedContents, password, csprng))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fs := &Filestore{
		basedir:  basedir,
		password: password,
		keyLocks: make(map[string]*sync.RWMutex),
		csprng:   csprng,
	}
	return fs, nil
}

// SetNonceGenerator sets the cryptographically secure pseudo-random
// number generator (csprng) used during encryption to generate nonces.
func (f *Filestore) SetNonceGenerator(csprng io.Reader) {
	f.csprng = csprng
}

// Close is equivalent to nil'ing out the Filestore object. This function
// is in place for the future when we add secure memory storage for keys.
func (f *Filestore) Close() {
	f.password = ""
	f.basedir = ""
	f.keyLocks = nil
	f.csprng = nil
}

// Set the value for the given key
func (f *Filestore) Set(key string, objectToStore Marshaler) error {
	return f.setData(key, objectToStore.Marshal())
}

// Get the value for the given key
func (f *Filestore) Get(key string, loadIntoThisObject Unmarshaler) error {
	decryptedContents, err := f.getData(key)
	if err == nil {
		err = loadIntoThisObject.Unmarshal(decryptedContents)
	}
	return errors.WithStack(err)
}

// Delete the value for the given key
func (f *Filestore) Delete(key string) error {
	encryptedKey := f.getKey(key)
	lck := f.getLock(encryptedKey)
	lck.Lock()
	defer lck.Unlock()
	return deleteFiles(encryptedKey, f.csprng)
}

// SetInterface uses json to encode and set data.
func (f *Filestore) SetInterface(key string, objectToStore interface{}) error {
	data, err := json.Marshal(objectToStore)
	if err == nil {
		err = f.setData(key, data)
	}
	return errors.WithStack(err)
}

// GetInterface uses json to encode and get data
func (f *Filestore) GetInterface(key string, v interface{}) error {
	data, err := f.getData(key)
	if err == nil {
		err = json.Unmarshal(data, v)
	}
	return errors.WithStack(err)
}

// Internal helper functions

func (f *Filestore) getLock(encryptedKey string) *sync.RWMutex {
	f.RLock()
	lck, ok := f.keyLocks[encryptedKey]
	f.RUnlock()
	if ok {
		return lck
	}
	// Note that 2 threads can get to this line at the same time,
	// which is why we check again after taking the write lock
	f.Lock()
	defer f.Unlock()

	lck, ok = f.keyLocks[encryptedKey]
	if ok {
		return lck
	}
	lck = &sync.RWMutex{}
	f.keyLocks[encryptedKey] = lck
	return lck
}

func (f *Filestore) getKey(key string) string {
	encryptedKey := hashStringWithPassword(key, f.password)
	encryptedKeyStr := hex.EncodeToString(encryptedKey)
	return f.basedir + string(os.PathSeparator) + encryptedKeyStr
}

func (f *Filestore) getData(key string) ([]byte, error) {
	encryptedKey := f.getKey(key)
	lck := f.getLock(encryptedKey)

	lck.RLock()
	encryptedContents, err := read(encryptedKey)
	lck.RUnlock()

	var decryptedContents []byte
	if err == nil {
		decryptedContents, err = decrypt(encryptedContents, f.password)
	}
	return decryptedContents, errors.WithStack(err)
}

func (f *Filestore) setData(key string, data []byte) error {
	encryptedKey := f.getKey(key)
	encryptedContents := encrypt(data, f.password, f.csprng)

	lck := f.getLock(encryptedKey)
	lck.Lock()
	defer lck.Unlock()

	err := write(encryptedKey, encryptedContents)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
