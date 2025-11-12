////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/ekv/portable"
)

const (
	kvDebugHeader = "[KV FILE DEBUG]"
)

// Filestore implements an ekv by reading and writing to files in a
// directory.
type Filestore struct {
	basedir  string
	password string
	sync.RWMutex
	keyLocks map[string]*sync.RWMutex
	csprng   io.Reader
	storage  portable.Storage
}

// NewFilestore returns an initialized filestore object or an error
// if it can't read and write to the directory/.ekv.1/2 file. Note that
// this file is not used other than to verify read/write capabilities on the
// directory. This uses the standard POSIX filesystem.
func NewFilestore(basedir, password string) (*Filestore, error) {
	return NewFilestoreWithNonceGenerator(basedir, password, rand.Reader)
}

// NewFilestoreWithNonceGenerator returns an initialized filestore object that
// uses a custom RNG for Nonce generation. This uses the standard POSIX filesystem.
func NewFilestoreWithNonceGenerator(basedir, password string,
	csprng io.Reader) (*Filestore, error) {
	return NewGenericFilestoreWithNonceGenerator(portable.UsePosix(), basedir, password, csprng)
}

// NewKeyValueFilestore returns an initialized filestore backed by a
// GenericKeyValue interface. This allows using any key-value store
// (e.g., browser localStorage, IndexedDB) as the storage backend.
func NewKeyValueFilestore(kv portable.GenericKeyValue, basedir, password string) (*Filestore, error) {
	return NewKeyValueFilestoreWithNonceGenerator(kv, basedir, password, rand.Reader)
}

// NewKeyValueFilestoreWithNonceGenerator returns an initialized filestore
// backed by a GenericKeyValue interface with a custom RNG for Nonce generation.
func NewKeyValueFilestoreWithNonceGenerator(kv portable.GenericKeyValue, basedir, password string,
	csprng io.Reader) (*Filestore, error) {
	return NewGenericFilestoreWithNonceGenerator(portable.UseKeyValue(kv), basedir, password, csprng)
}

// NewGenericFilestore returns an initialized filestore backed by a
// generic Storage interface.
func NewGenericFilestore(storage portable.Storage, basedir, password string) (*Filestore, error) {
	return NewGenericFilestoreWithNonceGenerator(storage, basedir, password, rand.Reader)
}

// NewGenericFilestoreWithNonceGenerator returns an initialized filestore
// backed by a generic Storage interface with a custom RNG for Nonce generation.
func NewGenericFilestoreWithNonceGenerator(storage portable.Storage, basedir, password string,
	csprng io.Reader) (*Filestore, error) {
	// Create the directory if it doesn't exist, otherwise do nothing.
	err := storage.MkdirAll(basedir, 0700)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Get the path to the "ekv" file
	ekvPath := basedir + string(os.PathSeparator) + ".ekv"
	expectedContents := []byte("version:1")

	// Try to read the .ekv.1/2 file, if it exists then we check
	// it's contents
	ekvCiphertext, err := read(ekvPath, storage)
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
	err = write(ekvPath, encrypt(expectedContents, password, csprng), storage)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fs := &Filestore{
		basedir:  basedir,
		password: password,
		keyLocks: make(map[string]*sync.RWMutex),
		csprng:   csprng,
		storage:  storage,
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
	unlock := f.takeWriteLock(encryptedKey)
	defer unlock()
	jww.TRACE.Printf("%s,DELETE,%s,%s", kvDebugHeader, key, encryptedKey)
	return deleteFiles(encryptedKey, f.csprng, f.storage)
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

// GetBytes implements [KeyValue.GetBytes]
func (f *Filestore) GetBytes(key string) ([]byte, error) {
	encryptedKey := f.getKey(key)
	unlock := f.takeReadLock(encryptedKey)

	encryptedContents, err := read(encryptedKey, f.storage)
	unlock()

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
	jww.TRACE.Printf(
		"%s,SET,%s,%s,%s", kvDebugHeader, key, encryptedKey, data)
	unlock := f.takeWriteLock(encryptedKey)
	defer unlock()

	err := write(encryptedKey, encryptedContents, f.storage)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Transaction implements [KeyValue.Transaction]
func (f *Filestore) Transaction(op TransactionOperation, keys ...string) error {

	// setup and get the data
	e := newExtendable(f)
	defer e.close()
	operables, err := e.Extend(keys)
	if err != nil {
		return err
	}

	// jww.TRACE.Printf(
	// 	"%s,TRANSACTION,%s,%s,%s", kvDebugHeader, key, encryptedKey, data)

	// do the operations
	err = op(operables, e)
	if err != nil {
		return err
	}

	// flush operations
	e.flush()

	return nil
}

// Internal helper functions

func (f *Filestore) takeWriteLock(encryptedKey string) (unlock func()) {
	f.RLock()
	lck, ok := f.keyLocks[encryptedKey]
	if ok {
		lck.Lock()
		f.RUnlock()
		unlock = lck.Unlock
		return unlock
	}
	f.RUnlock()

	// Note that 2 threads can get to this line at the same time,
	// which is why we check again after taking the write lock
	f.Lock()

	lck, ok = f.keyLocks[encryptedKey]
	if !ok {
		lck = &sync.RWMutex{}
		f.keyLocks[encryptedKey] = lck
	}
	lck.Lock()
	unlock = lck.Unlock
	f.Unlock()
	return unlock
}

func (f *Filestore) takeReadLock(encryptedKey string) (unlock func()) {
	f.RLock()
	lck, ok := f.keyLocks[encryptedKey]
	if ok {
		lck.RLock()
		f.RUnlock()
		unlock = lck.RUnlock
		return unlock
	}
	f.RUnlock()

	// Note that 2 threads can get to this line at the same time,
	// which is why we check again after taking the write lock
	f.Lock()

	lck, ok = f.keyLocks[encryptedKey]
	if !ok {
		lck = &sync.RWMutex{}
		f.keyLocks[encryptedKey] = lck
	}
	lck.RLock()
	unlock = lck.RUnlock
	f.Unlock()
	return unlock
}

func (f *Filestore) takeTransactionLocks(encryptedKeys []string) (unlock func()) {
	locks := make([]*sync.RWMutex, 0, len(encryptedKeys))

	f.Lock()

	for _, ecrKey := range encryptedKeys {
		lck, ok := f.keyLocks[ecrKey]
		if !ok {
			lck = &sync.RWMutex{}
			f.keyLocks[ecrKey] = lck
		}
		lck.Lock()
		locks = append(locks, lck)
	}

	f.Unlock()

	return func() {
		for _, lck := range locks {
			lck.Unlock()
		}
	}
}

type extendable struct {
	closed    bool
	unlock    func()
	f         *Filestore
	operables []map[string]Operable
}

func newExtendable(f *Filestore) *extendable {
	return &extendable{
		closed: false,
		unlock: func() {},
		f:      f,
	}
}

func (e *extendable) Extend(keys []string) (map[string]Operable, error) {
	if e.closed {
		jww.FATAL.Panicf("Cannot extend, transaction already closed")
	}
	operables := make(map[string]Operable, len(keys))
	ecrKeys := make([]string, len(keys))

	// make the ecrypted keys
	for i, key := range keys {
		ecrkey := e.f.getKey(key)
		operables[key] = &operable{
			key:    key,
			closed: false,
			ecrKey: ecrkey,
			op:     readOp,
			f:      e.f,
		}
		ecrKeys[i] = ecrkey
	}

	// get the locks
	e.addUnlock(e.f.takeTransactionLocks(ecrKeys))

	// read the keys
	for _, oper := range operables {
		operInternal := oper.(*operable)
		encryptedContents, err := read(operInternal.ecrKey, e.f.storage)
		// if an error is received which is not the file is not found, return it
		hasfile := true
		if err != nil {
			if !Exists(err) {
				hasfile = false
			} else {
				return nil, err
			}
		}

		var decryptedContents []byte
		if hasfile {
			decryptedContents, err = decrypt(encryptedContents, e.f.password)
			if err != nil {
				return nil, err
			}
		}
		operInternal.exists = hasfile
		operInternal.existed = hasfile
		operInternal.data = decryptedContents
	}
	e.operables = append(e.operables, operables)
	return operables, nil
}

func (e *extendable) IsClosed() bool {
	return e.closed
}

func (e *extendable) addUnlock(u func()) {
	oldUnlock := e.unlock
	e.unlock = func() {
		oldUnlock()
		u()
	}
}

func (e *extendable) flush() {
	for _, opMap := range e.operables {
		for _, oper := range opMap {
			if !oper.IsClosed() {
				if err := oper.Flush(); err != nil {
					jww.FATAL.Panicf("Failed on a flush of key %s in "+
						"transaction: %+v", oper.Key(), err)
				}
			}
		}
	}
}

func (e *extendable) close() {
	e.closed = true
	e.unlock()
}

type operable struct {
	key    string
	closed bool

	ecrKey string

	data    []byte
	exists  bool
	existed bool

	op OperableOps

	f *Filestore
}

func (op *operable) Key() string {
	op.testClosed("Key()")
	return op.key
}

func (op *operable) Exists() bool {
	op.testClosed("Exists()")
	return op.exists
}

func (op *operable) Delete() {
	op.testClosed("Delete()")

	op.data = nil
	op.exists = false
	op.op = deleteOp
}

func (op *operable) Set(data []byte) {
	op.testClosed("Set()")

	op.data = data
	op.exists = true
	op.op = writeOp
}

func (op *operable) Get() ([]byte, bool) {
	op.testClosed("Get()")
	return op.data, op.exists
}

func (op *operable) Flush() error {
	op.testClosed("Flush()")
	defer func() {
		op.closed = true
	}()
	switch op.op {
	case readOp:
		return nil
	case writeOp:
		encryptedNewContents := encrypt(op.data, op.f.password, op.f.csprng)
		return write(op.ecrKey, encryptedNewContents, op.f.storage)
	case deleteOp:
		if op.existed {
			return deleteFiles(op.ecrKey, op.f.csprng, op.f.storage)
		}
		return nil

	}
	return nil
}

func (op *operable) IsClosed() bool {
	return op.closed
}

func (op *operable) testClosed(action string) {
	if op.closed {
		jww.FATAL.Panicf("Cannot '%s' on '%s', already closed", action, op.key)
	}
}

type OperableOps uint8

const (
	readOp OperableOps = iota
	writeOp
	deleteOp
)

func (f *Filestore) getKey(key string) string {
	encryptedKey := hashStringWithPassword(key, f.password)
	encryptedKeyStr := encodeKey(encryptedKey)
	return f.basedir + string(os.PathSeparator) + encryptedKeyStr
}
