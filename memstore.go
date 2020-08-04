////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Elixxir                                                    /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"encoding/json"
	"github.com/pkg/errors"
)

// Memstore is an unencrypted memory based map that implements the KV interface
type Memstore map[string][]byte

// Set stores the value if there's no serialization error
func (m Memstore) Set(key string, objectToStore Marshaler) error {
	ser := objectToStore.Marshal()
	m[key] = ser
	return nil
}

// Get returns the value
func (m Memstore) Get(key string, loadIntoThisObject Unmarshaler) error {
	data, ok := m[key]
	if !ok {
		return errors.New("object not found")
	}
	return loadIntoThisObject.Unmarshal(data)
}

// Get returns the value
func (m Memstore) Delete(key string) error {
	delete(m, key)
	return nil
}

// SetInterface sets the value using a json encoder
func (m Memstore) SetInterface(key string, objectToStore interface{}) error {
	data, err := json.Marshal(objectToStore)
	if err != nil {
		return errors.Wrap(err, "SetInterface error")
	}
	m[key] = data
	return nil
}

// GetInterface gets the value using a json encoder
func (m Memstore) GetInterface(key string, objectToLoad interface{}) error {
	data, ok := m[key]
	if !ok {
		return errors.New("object not found")
	}
	return json.Unmarshal(data, objectToLoad)
}
