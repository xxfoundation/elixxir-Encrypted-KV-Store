////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"encoding/json"
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
func (m *Memstore) Transaction(key string, op TransactionOperation) (
	old []byte, existed bool, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	old, existed = m.store[key]

	var newData []byte
	newData, err = op(old, existed)
	if err != nil {
		return nil, existed, err
	}

	m.store[key] = newData

	return old, existed, nil
}
