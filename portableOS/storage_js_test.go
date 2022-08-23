///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// This file is only compiled for WebAssembly.

package portableOS

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// Tests that a value set with jsStore.setItem and retrieved with
// jsStore.getItem matches the original.
func Test_jsStore_getItem_setItem(t *testing.T) {
	values := map[string][]byte{
		"key1": []byte("key value"),
		"key2": {0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		"key3": {0, 49, 0, 0, 0, 38, 249, 93, 242, 189, 222, 32, 138, 248, 121,
			151, 42, 108, 82, 199, 163, 61, 4, 200, 140, 231, 225, 20, 35, 243,
			253, 161, 61, 2, 227, 208, 173, 183, 33, 66, 236, 107, 105, 119, 26,
			42, 44, 60, 109, 172, 38, 47, 220, 17, 129, 4, 234, 241, 141, 81,
			84, 185, 32, 120, 115, 151, 128, 196, 143, 117, 222, 78, 44, 115,
			109, 20, 249, 46, 158, 139, 231, 157, 54, 219, 141, 252},
	}

	for keyName, keyValue := range values {
		jsStorage.setItem(keyName, keyValue)

		loadedValue, err := jsStorage.getItem(keyName)
		if err != nil {
			t.Errorf("Failed to load %q: %+v", keyName, err)
		}

		if !bytes.Equal(keyValue, loadedValue) {
			t.Errorf("Loaded value does not match original for %q"+
				"\nexpected: %q\nreceived: %q", keyName, keyValue, loadedValue)
		}
	}
}

// Tests that jsStore.getItem returns the error os.ErrNotExist when the key does
// not exist in storage.
func Test_jsStore_getItem_NotExistError(t *testing.T) {
	_, err := jsStorage.getItem("someKey")
	if err == nil || !strings.Contains(err.Error(), os.ErrNotExist.Error()) {
		t.Errorf("Incorrect error for non existant key."+
			"\nexpected: %v\nreceived: %v", os.ErrNotExist, err)
	}
}

// Tests that jsStore.removeItem deletes a key from store and that it cannot be
// retrieved.
func Test_jsStore_removeItem(t *testing.T) {
	keyName := "key"
	jsStorage.setItem(keyName, []byte("value"))
	jsStorage.removeItem(keyName)

	_, err := jsStorage.getItem(keyName)
	if err == nil || !strings.Contains(err.Error(), os.ErrNotExist.Error()) {
		t.Errorf("Failed to remove %q: %+v", keyName, err)
	}
}

// Tests that jsStore.key return all added keys when looping through all
// indexes.
func Test_jsStore_key(t *testing.T) {
	jsStorage.Call("clear")
	values := map[string][]byte{
		"key1": []byte("key value"),
		"key2": {0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		"key3": {0, 49, 0, 0, 0, 38, 249, 93},
	}

	for keyName, keyValue := range values {
		jsStorage.setItem(keyName, keyValue)
	}

	numKeys := len(values)
	for i := 0; i < numKeys; i++ {
		keyName, err := jsStorage.key(i)
		if err != nil {
			t.Errorf("No key found for index %d: %+v", i, err)
		}

		if _, exists := values[keyName]; !exists {
			t.Errorf("No key with name %q added to storage.", keyName)
		}
		delete(values, keyName)
	}

	if len(values) != 0 {
		t.Errorf("%d keys not read from storage: %q", len(values), values)
	}
}

// Tests that jsStore.key returns the error os.ErrNotExist when the index is
// greater than or equal to the number of keys.
func Test_jsStore_key_NotExistError(t *testing.T) {
	jsStorage.Call("clear")
	jsStorage.setItem("key", []byte("value"))

	_, err := jsStorage.key(1)
	if err == nil || !strings.Contains(err.Error(), os.ErrNotExist.Error()) {
		t.Errorf("Incorrect error for non existant key index."+
			"\nexpected: %v\nreceived: %v", os.ErrNotExist, err)
	}

	_, err = jsStorage.key(2)
	if err == nil || !strings.Contains(err.Error(), os.ErrNotExist.Error()) {
		t.Errorf("Incorrect error for non existant key index."+
			"\nexpected: %v\nreceived: %v", os.ErrNotExist, err)
	}
}

// Tests that jsStore.length returns the correct length when adding and removing
// various keys.
func Test_jsStore_length(t *testing.T) {
	jsStorage.Call("clear")
	values := map[string][]byte{
		"key1": []byte("key value"),
		"key2": {0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		"key3": {0, 49, 0, 0, 0, 38, 249, 93},
	}

	i := 0
	for keyName, keyValue := range values {
		jsStorage.setItem(keyName, keyValue)
		i++

		if jsStorage.length() != i {
			t.Errorf("Incorrect length.\nexpected: %d\nreceived: %d",
				i, jsStorage.length())
		}
	}

	i = len(values)
	for keyName := range values {
		jsStorage.removeItem(keyName)
		i--

		if jsStorage.length() != i {
			t.Errorf("Incorrect length.\nexpected: %d\nreceived: %d",
				i, jsStorage.length())
		}
	}
}
