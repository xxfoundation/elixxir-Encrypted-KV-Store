///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// This file is only compiled for WebAssembly.

package portableOS

import (
	"encoding/base64"
	"os"
	"syscall/js"
)

// jsStore contains the js.Value representation of localStorage.
type jsStore struct {
	js.Value
}

// Defines storage used by Javascript as window.localStorage.
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-localstorage-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Window/localStorage
var jsStorage = jsStore{js.Global().Get("localStorage")}

// getItem returns a key's value from the local storage given its name. Returns
// os.ErrNotExist if the key does not exist. Underneath, it calls
// localStorage.getItem().
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-storage-getitem-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Storage/getItem
func (s *jsStore) getItem(keyName string) ([]byte, error) {
	keyValue := s.Call("getItem", keyName)
	if keyValue.IsNull() {
		return nil, os.ErrNotExist
	}

	decodedKeyValue, err := base64.StdEncoding.DecodeString(keyValue.String())
	if err != nil {
		return nil, err
	}

	return decodedKeyValue, nil
}

// setItem adds a key's value to local storage given its name. Underneath, it
// calls localStorage.setItem().
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-storage-setitem-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Storage/setItem
func (s *jsStore) setItem(keyName string, keyValue []byte) {
	encodedKeyValue := base64.StdEncoding.EncodeToString(keyValue)
	s.Call("setItem", keyName, encodedKeyValue)
}

// removeItem removes a key's value from local storage given its name. If there
// is no item with the given key, this function does nothing. Underneath, it
// calls localStorage.removeItem().
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-storage-removeitem-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Storage/removeItem
func (s *jsStore) removeItem(keyName string) {
	s.Call("removeItem", keyName)
}

// key returns the name of the nth key in localStorage. Return os.ErrNotExist if
// the key does not exist. The order of keys is not defined. If there is no item
// with the given key, this function does nothing. Underneath, it calls
// localStorage.key().
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-storage-key-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Storage/key
func (s *jsStore) key(n int) (string, error) {
	keyName := s.Call("key", n)
	if keyName.IsNull() {
		return "", os.ErrNotExist
	}

	return keyName.String(), nil
}

// length returns the number of keys in localStorage. Underneath, it accesses
// the property localStorage.length.
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-storage-key-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Storage/length
func (s *jsStore) length() int {
	return s.Get("length").Int()
}
