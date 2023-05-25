////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"sync"

	"github.com/pkg/errors"
)

const (
	objectNotFoundErr = "object not found"
	setInterfaceErr   = "SetInterface error"
)

// Memstore is an unencrypted memory-based map that implements the KeyValue
// interface.
type Memstore struct {
	store map[string][]byte
	mux   sync.RWMutex
}

// MakeMemstore returns a new Memstore with a newly initialised a new map.
func MakeMemstore() *Memstore {
	return &Memstore{store: make(map[string][]byte)}
}

// Set stores the value if there's no serialization error per [KeyValue.Set]
func (m *Memstore) Set(key string, objectToStore Marshaler) error {
	return m.SetBytes(key, objectToStore.Marshal())
}

// Get implements [KeyValue.Get]
func (m *Memstore) Get(key string, loadIntoThisObject Unmarshaler) error {
	data, err := m.GetBytes(key)
	if err != nil {
		return err
	}
	return loadIntoThisObject.Unmarshal(data)
}

// Delete removes the value from the store per [KeyValue.Delete]
func (m *Memstore) Delete(key string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	delete(m.store, key)
	return nil
}

// SetInterface sets the value using a JSON encoder per [KeyValue.SetInterface]
func (m *Memstore) SetInterface(key string, objectToStore interface{}) error {
	data, err := json.Marshal(objectToStore)
	if err != nil {
		return errors.Wrap(err, setInterfaceErr)
	}
	return m.SetBytes(key, data)
}

// GetInterface gets the value using a JSON encoder per [KeyValue.GetInterface]
func (m *Memstore) GetInterface(key string, objectToLoad interface{}) error {
	data, err := m.GetBytes(key)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, objectToLoad)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// SetBytes implements [KeyValue.SetBytes]
func (m *Memstore) SetBytes(key string, data []byte) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.store[key] = data
	return nil
}

// SetBytes implements [KeyValue.GetBytes]
func (m *Memstore) GetBytes(key string) ([]byte, error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	data, ok := m.store[key]
	if !ok {
		return nil, errors.New(objectNotFoundErr)
	}

	return data, nil
}

// Transaction implements [KeyValue.Transaction]
func (m *Memstore) Transaction(op TransactionOperation, keys ...string) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	e := &extendableMem{
		closed: false,
		mem:    m,
	}
	defer e.close()

	operables, err := e.Extend(keys)
	if err != nil {
		return err
	}

	err = op(operables, e)
	if err != nil {
		return err
	}

	e.flush()
	return nil
}

type extendableMem struct {
	closed    bool
	mem       *Memstore
	operables []map[string]Operable
}

func (e *extendableMem) Extend(keys []string) (map[string]Operable, error) {
	if e.closed {
		jww.FATAL.Panicf("Cannot extend, transaction already closed")
	}
	operables := make(map[string]Operable, len(keys))

	// make the ecrypted keys
	for _, key := range keys {
		operables[key] = &operableMem{
			key:    key,
			closed: false,
			op:     readOp,
			mem:    e.mem,
		}
	}

	// read the keys
	for _, oper := range operables {
		operInternal := oper.(*operableMem)
		operInternal.data, operInternal.exists = e.mem.store[operInternal.key]
	}
	e.operables = append(e.operables, operables)
	return operables, nil
}

func (e *extendableMem) IsClosed() bool {
	return e.closed
}

func (e *extendableMem) flush() {
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

func (e *extendableMem) close() {
	e.closed = true
}

type operableMem struct {
	key    string
	closed bool

	data   []byte
	exists bool

	op OperableOps

	mem *Memstore
}

func (op *operableMem) Key() string {
	op.testClosed("Key()")
	return op.key
}

func (op *operableMem) Exists() bool {
	op.testClosed("Exists()")
	return op.exists
}

func (op *operableMem) Delete() {
	op.testClosed("Delete()")

	op.data = nil
	op.exists = false
	op.op = deleteOp
}

func (op *operableMem) Set(data []byte) {
	op.testClosed("Set()")

	op.data = data
	op.exists = true
	op.op = writeOp
}

func (op *operableMem) Get() ([]byte, bool) {
	op.testClosed("Get()")
	return op.data, op.exists
}

func (op *operableMem) Flush() error {
	op.testClosed("Flush()")
	defer func() {
		op.closed = true
		op.mem.mux.Unlock()
	}()
	switch op.op {
	case readOp:
		return nil
	case writeOp:
		op.mem.store[op.key] = op.data
	case deleteOp:
		delete(op.mem.store, op.key)
	}
	return nil
}

func (op *operableMem) IsClosed() bool {
	return op.closed
}

func (op *operableMem) testClosed(action string) {
	if op.closed {
		jww.FATAL.Panicf("Cannot '%s' on '%s', already closed", action, op.key)
	}
}
