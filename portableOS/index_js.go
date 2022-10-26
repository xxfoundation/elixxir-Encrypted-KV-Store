////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// This file is only compiled for WebAssembly.

package portableOS

import (
	"context"
	"github.com/pkg/errors"
	"syscall/js"
	"time"

	"github.com/hack-pad/go-indexeddb/idb"
	jww "github.com/spf13/jwalterweatherman"
)

const (
	// databaseName is the name of the [idb.Database].
	databaseName = "ekv"

	// currentVersion is the current version of the IndexDb
	// runtime. Used for migration purposes.
	currentVersion uint = 1

	// Text representation of primary key value (keyPath).
	pkeyName = "id"

	// Text representation of the names of the [idb.ObjectStore].
	stateStoreName = "state"

	// dbTimeout is the global timeout for operations with the storage
	// [context.Context].
	dbTimeout = time.Second
)

// indexStore contains the js.Value representation of localStorage.
type indexStore struct {
	db *idb.Database
}

var jsDb *indexStore

func init() {
	var err error
	jsDb, err = newIndexStore()
	if err != nil {
		jww.FATAL.Panicf("Failed to initialise indexedDb: %+v", err)
	}
}

// newIndexStore creates the [idb.Database] and returns a wasmModel.
func newIndexStore() (*indexStore, error) {
	// Attempt to open database object
	ctx, cancel := newContext()
	defer cancel()
	openRequest, err := idb.Global().Open(ctx, databaseName, currentVersion,
		func(db *idb.Database, oldVersion, newVersion uint) error {
			if oldVersion == newVersion {
				jww.INFO.Printf("IndexDb %s version is current: v%d",
					databaseName, newVersion)
				return nil
			}

			jww.INFO.Printf("IndexDb %s upgrade required: v%d -> v%d",
				databaseName, oldVersion, newVersion)

			if oldVersion == 0 && newVersion >= 1 {
				err := v1Upgrade(db)
				if err != nil {
					return err
				}
				oldVersion = 1
			}

			// if oldVersion == 1 && newVersion >= 2 { v2Upgrade(), oldVersion = 2 }
			return nil
		})
	if err != nil {
		return nil, err
	}

	// Wait for database open to finish
	db, err := openRequest.Await(ctx)
	return &indexStore{db: db}, err
}

// v1Upgrade performs the v0 -> v1 database upgrade.
//
// This can never be changed without permanently breaking backwards
// compatibility.
func v1Upgrade(db *idb.Database) error {
	storeOpts := idb.ObjectStoreOptions{
		KeyPath:       js.ValueOf(pkeyName),
		AutoIncrement: false,
	}

	// Build Message ObjectStore and Indexes
	_, err := db.CreateObjectStore(stateStoreName, storeOpts)
	return err
}

// newContext builds a context for database operations.
func newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), dbTimeout)
}

// getItem returns a key's value from the local storage given its name. Returns
// os.ErrNotExist if the key does not exist. Underneath, it calls
// localStorage.getItem().
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-storage-getitem-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Storage/getItem
func (s *indexStore) getItem(keyName string) ([]byte, error) {
	parentErr := errors.New("failed to getItem")

	// Prepare the Transaction
	txn, err := s.db.Transaction(idb.TransactionReadWrite, stateStoreName)
	if err != nil {
		return nil, errors.WithMessagef(parentErr,
			"Unable to create Transaction: %+v", err)
	}
	store, err := txn.ObjectStore(stateStoreName)
	if err != nil {
		return nil, errors.WithMessagef(parentErr,
			"Unable to get ObjectStore: %+v", err)
	}

	// Perform the operation
	getRequest, err := store.Get(CopyBytesToJS([]byte(keyName)))
	if err != nil {
		return nil, errors.WithMessagef(parentErr,
			"Unable to Get: %+v", err)
	}

	// Wait for the operation to return
	ctx, cancel := newContext()
	resultObj, err := getRequest.Await(ctx)
	cancel()
	if err != nil {
		return nil, errors.WithMessagef(parentErr,
			"Unable to get from ObjectStore: %+v", err)
	}

	jww.DEBUG.Printf("Got from %s/%s", stateStoreName, keyName)
	return []byte(resultObj.String()), nil
}

// setItem adds a key's value to local storage given its name. Underneath, it
// calls localStorage.setItem().
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-storage-setitem-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Storage/setItem
func (s *indexStore) setItem(keyName string, keyValue []byte) {
	parentErr := errors.New("failed to setItem")

	// Prepare the Transaction
	txn, err := s.db.Transaction(idb.TransactionReadWrite, stateStoreName)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to create Transaction: %+v", err))
		return
	}
	store, err := txn.ObjectStore(stateStoreName)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to get ObjectStore: %+v", err))
		return
	}

	// Perform the operation
	_, err = store.PutKey(CopyBytesToJS([]byte(keyName)), CopyBytesToJS(keyValue))
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to Put Key: %+v", err))
		return
	}

	// Wait for the operation to return
	ctx, cancel := newContext()
	err = txn.Await(ctx)
	cancel()
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"setItem failed: %+v", err))
		return
	}
	jww.DEBUG.Printf("Successful setItem: %s", keyName)
}

// removeItem removes a key's value from local storage given its name. If there
// is no item with the given key, this function does nothing. Underneath, it
// calls localStorage.removeItem().
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-storage-removeitem-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Storage/removeItem
func (s *indexStore) removeItem(keyName string) {
	parentErr := errors.New("failed to removeItem")

	// Prepare the Transaction
	txn, err := s.db.Transaction(idb.TransactionReadWrite, stateStoreName)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to create Transaction: %+v", err))
		return
	}
	store, err := txn.ObjectStore(stateStoreName)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to get ObjectStore: %+v", err))
		return
	}

	// Perform the operation
	_, err = store.Delete(CopyBytesToJS([]byte(keyName)))
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to Delete Key: %+v", err))
		return
	}

	// Wait for the operation to return
	ctx, cancel := newContext()
	err = txn.Await(ctx)
	cancel()
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"removeItem failed: %+v", err))
		return
	}
	jww.DEBUG.Printf("Successful removeItem: %s", keyName)
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
func (s *indexStore) key(n int) (string, error) {
	parentErr := errors.Errorf("failed to get key")

	txn, err := s.db.Transaction(idb.TransactionReadOnly, stateStoreName)
	if err != nil {
		return "", errors.WithMessagef(parentErr,
			"Unable to create Transaction: %+v", err)
	}
	store, err := txn.ObjectStore(stateStoreName)
	if err != nil {
		return "", errors.WithMessagef(parentErr,
			"Unable to get ObjectStore: %+v", err)
	}
	cursorRequest, err := store.OpenCursor(idb.CursorNext)
	if err != nil {
		return "", errors.WithMessagef(parentErr,
			"Unable to build Cursor: %+v", err)
	}

	// Run the query
	ctx, cancel := newContext()
	cursor, err := cursorRequest.Await(ctx)
	if err != nil {
		return "", err
	}
	cancel()
	if err != nil {
		return "", errors.WithMessagef(parentErr,
			"Unable to open Cursor: %+v", err)
	}

	// Advance the cursor and return its value
	err = cursor.Advance(uint(n))
	if err != nil {
		return "", errors.WithMessagef(parentErr,
			"Unable to advance Cursor: %+v", err)
	}
	value, err := cursor.Value()
	if err != nil {
		return "", errors.WithMessagef(parentErr,
			"Unable to get Cursor value: %+v", err)
	}
	return value.String(), nil
}

// length returns the number of keys in localStorage. Underneath, it accesses
// the property localStorage.length.
//
//  - Specification:
//    https://html.spec.whatwg.org/multipage/webstorage.html#dom-storage-key-dev
//  - Documentation:
//    https://developer.mozilla.org/en-US/docs/Web/API/Storage/length
func (s *indexStore) length() int {
	parentErr := errors.New("failed to length")

	// Prepare the Transaction
	txn, err := s.db.Transaction(idb.TransactionReadWrite, stateStoreName)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to create Transaction: %+v", err))
		return 0
	}
	store, err := txn.ObjectStore(stateStoreName)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to get ObjectStore: %+v", err))
		return 0
	}

	// Perform the operation
	countRequest, err := store.Count()
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to Count: %+v", err))
		return 0
	}

	// Wait for the operation to return
	ctx, cancel := newContext()
	countResult, err := countRequest.Await(ctx)
	cancel()
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to get from ObjectStore: %+v", err))
		return 0
	}

	jww.DEBUG.Printf("Successful length: %d", countResult)
	return int(countResult)
}

// Uint8Array is the Javascript Uint8Array object. It is used to create new
// Uint8Array.
var Uint8Array = js.Global().Get("Uint8Array")

// CopyBytesToGo copies the [Uint8Array] stored in the [js.Value] to []byte.
// This is a wrapper for [js.CopyBytesToGo] to make it more convenient.
func CopyBytesToGo(src js.Value) []byte {
	b := make([]byte, src.Length())
	js.CopyBytesToGo(b, src)
	return b
}

// CopyBytesToJS copies the []byte to a [Uint8Array] stored in a [js.Value].
// This is a wrapper for [js.CopyBytesToJS] to make it more convenient.
func CopyBytesToJS(src []byte) js.Value {
	dst := Uint8Array.New(len(src))
	js.CopyBytesToJS(dst, src)
	return dst
}
