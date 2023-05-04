////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/ekv/portableOS"

	jww "github.com/spf13/jwalterweatherman"
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

// Set the value for the given key per [KeyValue.Set]
func (f *Filestore) Set(key string, objectToStore Marshaler) error {
	return f.SetBytes(key, objectToStore.Marshal())
}

// Get the value for the given key per [KeyValue.Get]
func (f *Filestore) Get(key string, loadIntoThisObject Unmarshaler) error {
	decryptedContents, err := f.GetBytes(key)
	if err == nil {
		err = loadIntoThisObject.Unmarshal(decryptedContents)
	}
	return errors.WithStack(err)
}

// Delete the value for the given key per [KeyValue.Delete]
func (f *Filestore) Delete(key string) error {
	encryptedKey := f.getKey(key)
	lck := f.getLock(encryptedKey)
	lck.Lock()
	defer lck.Unlock()
	return deleteFiles(encryptedKey, f.csprng)
}

// SetInterface uses json to encode and set data per [KeyValue.SetInterface]
func (f *Filestore) SetInterface(key string, objectToStore interface{}) error {
	data, err := json.Marshal(objectToStore)
	if err == nil {
		err = f.SetBytes(key, data)
	}
	return errors.WithStack(err)
}

// GetInterface uses json to encode and get data per [KeyValue.GetInterface]
func (f *Filestore) GetInterface(key string, v interface{}) error {
	data, err := f.GetBytes(key)
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

// GetBytes implements [KeyValue.GetBytes]
func (f *Filestore) GetBytes(key string) ([]byte, error) {
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

// SetBytes implements [KeyValue.SetBytes]
func (f *Filestore) SetBytes(key string, data []byte) error {
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

// Transaction implements [KeyValue.Transaction]
func (f *Filestore) Transaction(key string, op TransactionOperation) (
	old []byte, existed bool, err error) {
	encryptedKey := f.getKey(key)

	lck := f.getLock(encryptedKey)
	lck.Lock()
	defer lck.Unlock()

	//get the key
	encryptedContents, err := read(encryptedKey)
	// if an error is received which is not the file is not found, return it
	hasfile := true
	if err != nil {
		if !Exists(err) {
			hasfile = false
		} else {
			return nil, false, err
		}
	}
	var decryptedContents []byte
	if hasfile {
		decryptedContents, err = decrypt(encryptedContents, f.password)
		if err != nil {
			return nil, true, err
		}
	}

	data, deletion, err := op(decryptedContents, hasfile)
	if err != nil {
		return decryptedContents, hasfile, err
	}

	if deletion {
		err = deleteFile(encryptedKey, f.csprng)
		return decryptedContents, hasfile, err
	}

	encryptedNewContents := encrypt(data, f.password, f.csprng)

	err = write(encryptedKey, encryptedNewContents)
	if err != nil {
		return decryptedContents, hasfile, errors.WithStack(err)
	}
	return decryptedContents, hasfile, err
}

// MutualTransaction implements [KeyValue.MutualTransaction]
func (f *Filestore) MutualTransaction(keys []string,
	op MutualTransactionOperation) (map[string]Value, map[string]Value, error) {

	//get all keys - map of key to encrypted key
	encryptedKeys := make(map[string]string, len(keys))

	for _, key := range keys {
		encryptedKeys[key] = f.getKey(key)
	}

	//lock all key's locks
	for _, encryptedKey := range keys {
		lck := f.getLock(encryptedKey)
		lck.Lock()
		defer lck.Unlock()
	}

	//read each file
	oldContents := make(map[string]Value, len(keys))
	for key, encryptedKey := range encryptedKeys {
		encryptedContents, err := read(encryptedKey)
		hasfile := true
		if err != nil {
			if !Exists(err) {
				hasfile = false
			} else {
				return nil, nil, errors.WithMessagef(err,
					"Failed on loading from key %s", key)
			}
		}
		var decryptedContents []byte
		if hasfile {
			decryptedContents, err = decrypt(encryptedContents, f.password)
			if err != nil {
				return nil, nil, errors.WithMessagef(err,
					"Failed to decrypt from key %s", key)
			}
		}
		oldContents[key] = Value{
			Data:   decryptedContents,
			Exists: hasfile,
		}
	}

	//execute the op
	data, err := op(oldContents)

	if err != nil {
		return oldContents, nil, errors.WithMessagef(err,
			"Failed to execute transaction due to op failure")
	}

	// encrypt the data and write
	// note: operations are ordered per the incoming key list so
	// dependent keys can be put later in order to make it more likely
	// the system will operate if a write failure occurs
	deletions := make([]string, 0, len(keys))
	for _, key := range keys {
		v := data[key]
		if v.Exists {
			toWrite := encrypt(v.Data, f.password, f.csprng)
			err = write(encryptedKeys[key], toWrite)
			if err != nil {
				jww.FATAL.Panicf("Failed to write key %s to disk: %+v")
			}
		} else {
			deletions = append(deletions, key)
		}
	}

	// execute all deletions
	deletionFailure := false
	for _, key := range deletions {
		err = deleteFiles(encryptedKeys[key], f.csprng)
		if err != nil {
			data[key] = Value{
				Data:   nil,
				Exists: true,
			}
			jww.WARN.Printf("Deletion Failed for key %s: %+v\n",
				key, err)
			deletionFailure = true
		}
	}

	if deletionFailure {
		return oldContents, data, errors.New(ErrDeletesFailed)
	}

	return oldContents, data, nil
}

func (f *Filestore) getKey(key string) string {
	encryptedKey := hashStringWithPassword(key, f.password)
	encryptedKeyStr := hex.EncodeToString(encryptedKey)
	return f.basedir + string(os.PathSeparator) + encryptedKeyStr
}
