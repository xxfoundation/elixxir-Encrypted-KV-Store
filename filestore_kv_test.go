////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/ekv/portable"
)

// memoryKV is a simple in-memory implementation of portable.GenericKeyValue for testing
type memoryKV struct {
	data map[string][]byte
	mux  sync.RWMutex
}

func newMemoryKV() *memoryKV {
	return &memoryKV{
		data: make(map[string][]byte),
	}
}

func (m *memoryKV) Get(key string) ([]byte, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	val, ok := m.data[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	// Return a copy to avoid mutation
	result := make([]byte, len(val))
	copy(result, val)
	return result, nil
}

func (m *memoryKV) Set(key string, value []byte) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	// Store a copy to avoid mutation
	stored := make([]byte, len(value))
	copy(stored, value)
	m.data[key] = stored
	return nil
}

func (m *memoryKV) Delete(key string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	delete(m.data, key)
	return nil
}

func (m *memoryKV) Keys() ([]string, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

// TestFilestoreKV_Smoke runs a basic read/write using key-value backend
func TestFilestoreKV_Smoke(t *testing.T) {
	kv := newMemoryKV()

	f, err := NewKeyValueFilestore(kv, ".ekv_testdir_kv", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	i := &MarshalableString{
		S: "Hi",
	}
	err = f.Set("TestMe123", i)
	if err != nil {
		t.Error(err)
	}

	s := &MarshalableString{}
	err = f.Get("TestMe123", s)
	if err != nil {
		t.Error(err)
	}
	if s.S != "Hi" {
		t.Errorf("Did not get what we wrote: %s != %s", s.S, "Hi")
	}

	// Now test set/get Interface
	err = f.SetInterface("Test456", i)
	if err != nil {
		t.Error(err)
	}
	s = &MarshalableString{}
	err = f.GetInterface("Test456", s)
	if err != nil {
		t.Error(err)
	}
	if s.S != "Hi" {
		t.Errorf("Did not get what we wrote: %s != %s", s.S, "Hi")
	}

	err = f.Delete("Test456")
	if err != nil {
		t.Error(err)
	}
}

// TestFilestoreKV_Broken tries to marshal with a broken object
func TestFilestoreKV_Broken(t *testing.T) {
	kv := newMemoryKV()

	f, err := NewKeyValueFilestore(kv, ".ekv_testdir_kv_broken", "Hello, World 22!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	i := &BrokenMarshalable{
		S: "Hi",
	}
	err = f.Set("TestMe123", i)
	if err != nil {
		t.Error(err)
	}

	s := &BrokenMarshalable{}
	err = f.Get("TestMe123", s)
	if err == nil {
		t.Errorf("Unmarshal succeeded!")
	}
}

// TestFilestoreKV_Multiset makes sure we can continuously set the object and get
// the right result each time (exercises the internal monotonic counter functionality)
func TestFilestoreKV_Multiset(t *testing.T) {
	kv := newMemoryKV()

	f, err := NewKeyValueFilestore(kv, ".ekv_testdir_kv_multiset", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	for x := 0; x < 20; x++ {
		expStr := fmt.Sprintf("Hi, %d!", x)
		i := &MarshalableString{
			S: expStr,
		}
		err = f.Set("TestMe123", i)
		if err != nil {
			t.Error(err)
		}
		s := &MarshalableString{}
		err = f.Get("TestMe123", s)
		if err != nil {
			t.Error(err)
		}
		if s.S != expStr {
			t.Errorf("Did not get what we wrote: %s != %s", s.S,
				expStr)
		}
		// Now test set/get Interface
		err = f.SetInterface("Test456", i)
		if err != nil {
			t.Error(err)
		}
		s = &MarshalableString{}
		err = f.GetInterface("Test456", s)
		if err != nil {
			t.Error(err)
		}
		if s.S != expStr {
			t.Errorf("Did not get what we wrote: %s != %s", s.S,
				expStr)
		}
	}
}

// TestFilestoreKV_Reopen verifies we can recreate/reopen the store and get the
// data we stored back out using the same KV instance.
func TestFilestoreKV_Reopen(t *testing.T) {
	kv := newMemoryKV()

	f, err := NewKeyValueFilestore(kv, ".ekv_testdir_kv_reopen", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	expStr := "Hi"

	i := &MarshalableString{
		S: expStr,
	}
	err = f.Set("TestMe123", i)
	if err != nil {
		t.Error(err)
	}
	// Now test set/get Interface
	err = f.SetInterface("Test456", i)
	if err != nil {
		t.Error(err)
	}

	for x := 0; x < 20; x++ {
		// Reopen with the same KV instance to verify persistence
		f, err = NewKeyValueFilestore(kv, ".ekv_testdir_kv_reopen", "Hello, World!")
		if err != nil {
			t.Errorf("%+v", err)
		}

		s := &MarshalableString{}
		err = f.Get("TestMe123", s)
		if err != nil {
			t.Error(err)
		}
		if s.S != expStr {
			t.Errorf("Did not get what we wrote: %s != %s", s.S,
				expStr)
		}

		// Now test set/get Interface
		s = &MarshalableString{}
		err = f.GetInterface("Test456", s)
		if err != nil {
			t.Error(err)
		}
		if s.S != expStr {
			t.Errorf("Did not get what we wrote: %s != %s", s.S,
				expStr)
		}

		expStr = fmt.Sprintf("Hi, %d!", x)
		i := &MarshalableString{
			S: expStr,
		}
		err = f.Set("TestMe123", i)
		if err != nil {
			t.Error(err)
		}
		// Now test set/get Interface
		err = f.SetInterface("Test456", i)
		if err != nil {
			t.Error(err)
		}
	}
}

// TestFilestoreKV_BadPass confirms using a bad password nets an error
func TestFilestoreKV_BadPass(t *testing.T) {
	kv := newMemoryKV()

	_, err := NewKeyValueFilestore(kv, ".ekv_testdir_kv_badpass", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	_, err = NewKeyValueFilestore(kv, ".ekv_testdir_kv_badpass", "badpassword")
	if err == nil {
		t.Errorf("Opened with bad password!")
	}
}

// TestFilestoreKV_Concurrent tests concurrent access to different keys
func TestFilestoreKV_Concurrent(t *testing.T) {
	kv := newMemoryKV()

	f, err := NewKeyValueFilestore(kv, ".ekv_testdir_kv_concurrent", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	totalCnt := 50
	sharedCh := make(chan bool, totalCnt*2)

	for x := 0; x < totalCnt; x++ {
		// Kick off a read/write to a unique key
		go func(f *Filestore, x int) {
			expStr := fmt.Sprintf("Hi, %d!", x)
			keyStr := fmt.Sprintf("UniqueKey%d", x)
			i := &MarshalableString{
				S: expStr,
			}
			err := f.Set(keyStr, i)
			if err != nil {
				t.Error(err)
			}
			s := &MarshalableString{}
			err = f.Get(keyStr, s)
			if err != nil {
				t.Error(err)
			}
			if s.S != expStr {
				t.Errorf("Did not get what we wrote: %s != %s",
					s.S, expStr)
			}
			sharedCh <- true
			f.Delete(keyStr)
		}(f, x)

		// Kick off a read/write to the same key
		go func(f *Filestore, x int) {
			expStr := "Hi!"
			i := &MarshalableString{
				S: expStr,
			}
			keyStrInt := "SameKey"
			err := f.SetInterface(keyStrInt, i)
			if err != nil {
				t.Error(err)
			}
			s := &MarshalableString{}
			err = f.GetInterface(keyStrInt, s)
			if err != nil {
				t.Error(err)
			}
			if s.S != expStr {
				t.Errorf("Did not get what we wrote: %s != %s",
					s.S, expStr)
			}
			sharedCh <- true
		}(f, x)
	}

	finishedCnt := 0
	for finishedCnt < totalCnt*2 {
		<-sharedCh
		finishedCnt++
	}
}

// TestFilestoreKV_UseKeyValue tests creating a filestore directly with UseKeyValue
func TestFilestoreKV_UseKeyValue(t *testing.T) {
	kv := newMemoryKV()
	storage := portable.UseKeyValue(kv)

	f, err := NewGenericFilestore(storage, ".ekv_testdir_kv_generic", "TestPassword")
	if err != nil {
		t.Errorf("%+v", err)
	}

	i := &MarshalableString{S: "TestValue"}
	err = f.Set("TestKey", i)
	if err != nil {
		t.Error(err)
	}

	s := &MarshalableString{}
	err = f.Get("TestKey", s)
	if err != nil {
		t.Error(err)
	}
	if s.S != "TestValue" {
		t.Errorf("Did not get what we wrote: %s != %s", s.S, "TestValue")
	}
}

// TestMemoryKV_Interface verifies the memoryKV implementation works correctly
func TestMemoryKV_Interface(t *testing.T) {
	kv := newMemoryKV()

	// Test Set and Get
	err := kv.Set("key1", []byte("value1"))
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	val, err := kv.Get("key1")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if string(val) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", string(val))
	}

	// Test Get non-existent key
	_, err = kv.Get("nonexistent")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist, got %v", err)
	}

	// Test Keys
	kv.Set("key2", []byte("value2"))
	kv.Set("key3", []byte("value3"))

	keys, err := kv.Keys()
	if err != nil {
		t.Errorf("Keys failed: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Test Delete
	err = kv.Delete("key1")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	_, err = kv.Get("key1")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected key to be deleted")
	}

	keys, err = kv.Keys()
	if err != nil {
		t.Errorf("Keys failed: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys after delete, got %d", len(keys))
	}
}
