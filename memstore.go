////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"encoding/json"
	"github.com/pkg/errors"
	"sync"
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

// Set stores the value if there's no serialization error.
func (m *Memstore) Set(key string, objectToStore Marshaler) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	ser := objectToStore.Marshal()
	m.store[key] = ser
	return nil
}

// Get returns the value.
func (m *Memstore) Get(key string, loadIntoThisObject Unmarshaler) error {
	m.mux.RLock()
	defer m.mux.RUnlock()

	data, ok := m.store[key]
	if !ok {
		return errors.New(objectNotFoundErr)
	}
	return loadIntoThisObject.Unmarshal(data)
}

// Delete removes the value from the store.
func (m *Memstore) Delete(key string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	delete(m.store, key)
	return nil
}

// SetInterface sets the value using a JSON encoder.
func (m *Memstore) SetInterface(key string, objectToStore interface{}) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	data, err := json.Marshal(objectToStore)
	if err != nil {
		return errors.Wrap(err, setInterfaceErr)
	}

	m.store[key] = data
	return nil
}

// GetInterface gets the value using a JSON encoder.
func (m *Memstore) GetInterface(key string, objectToLoad interface{}) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	data, ok := m.store[key]
	if !ok {
		return errors.New(objectNotFoundErr)
	}

	err := json.Unmarshal(data, objectToLoad)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
