///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"gitlab.com/elixxir/ekv/portableOS"
	"testing"
)

// Tests happy path of Exists() with Memstore.
func TestExists_Memstore(t *testing.T) {
	f := MakeMemstore()
	i := &MarshalableString{S: "Hi"}
	err := f.Set("key2", i)
	if err != nil {
		t.Fatalf("Failed to save %s: %v", "key2", err)
	}

	err = f.Get("key1", nil)
	if Exists(err) {
		t.Errorf("Exists() reported the key exists: %v", err)
	}

	s := &MarshalableString{}
	err = f.Get("key2", s)
	if !Exists(err) {
		t.Errorf("Exists() did not report that the key exists: %v", err)
	}
}

// Tests happy path of Exists() with Filestore.
func TestExists_Filestore(t *testing.T) {
	dir := ".ekv_testdir"
	// Delete the test file at the end
	defer func() {
		err := portableOS.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error deleting test file %#v:\n%v", dir, err)
		}
	}()

	f, err := NewFilestore(dir, "Hello, World!")
	if err != nil {
		t.Fatalf("Failed to create filestore: %v", err)
	}

	i := &MarshalableString{S: "Hi"}
	err = f.Set("key2", i)
	if err != nil {
		t.Fatalf("Failed to save %s: %v", "key2", err)
	}

	err = f.Get("key1", nil)
	if Exists(err) {
		t.Errorf("Exists() reported the key exists: %v", err)
	}

	s := &MarshalableString{}
	err = f.Get("key2", s)
	if !Exists(err) {
		t.Errorf("Exists() did not report that the key exists: %v", err)
	}

}
