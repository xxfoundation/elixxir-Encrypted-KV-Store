///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"fmt"
	"testing"
)

// TestFilestore_Smoke runs a basic read/write on the current directory
func TestMemstore_Smoke(t *testing.T) {
	f := make(Memstore)
	i := &MarshalableString{
		S: "Hi",
	}
	err := f.Set("TestMe123", i)
	if err != nil {
		t.Errorf(err.Error())
	}

	s := &MarshalableString{}
	err = f.Get("TestMe123", s)
	if err != nil {
		t.Errorf(err.Error())
	}
	if s.S != "Hi" {
		t.Errorf("Did not get what we wrote: %s != %s", s.S, "Hi")
	}
	// Now test set/get Interface
	err = f.SetInterface("Test456", i)
	if err != nil {
		t.Errorf(err.Error())
	}
	s = &MarshalableString{}
	err = f.GetInterface("Test456", s)
	if err != nil {
		t.Errorf(err.Error())
	}
	if s.S != "Hi" {
		t.Errorf("Did not get what we wrote: %s != %s", s.S, "Hi")
	}
}

// TestFilestore_Broken tries to marshal with a broken object
func TestMemstore_Broken(t *testing.T) {
	f := make(Memstore)

	i := &BrokenMarshalable{
		S: "Hi",
	}
	err := f.Set("TestMe123", i)
	if err != nil {
		t.Errorf(err.Error())
	}

	s := &BrokenMarshalable{}
	err = f.Get("TestMe123", s)
	if err == nil {
		t.Errorf("Unmarshal succeded!")
	}
}

// TestFilestore_Multiset makes sure we can continuously set the object and get
// the right result each time (exercises the internal monotonic counter
// functionality)
func TestMemstore_Multiset(t *testing.T) {
	f := make(Memstore)

	for x := 0; x < 20; x++ {
		expStr := fmt.Sprintf("Hi, %d!", x)
		i := &MarshalableString{
			S: expStr,
		}
		err := f.Set("TestMe123", i)
		if err != nil {
			t.Errorf(err.Error())
		}
		s := &MarshalableString{}
		err = f.Get("TestMe123", s)
		if err != nil {
			t.Errorf(err.Error())
		}
		if s.S != expStr {
			t.Errorf("Did not get what we wrote: %s != %s", s.S,
				expStr)
		}
		// Now test set/get Interface
		err = f.SetInterface("Test456", i)
		if err != nil {
			t.Errorf(err.Error())
		}
		s = &MarshalableString{}
		err = f.GetInterface("Test456", s)
		if err != nil {
			t.Errorf(err.Error())
		}
		if s.S != expStr {
			t.Errorf("Did not get what we wrote: %s != %s", s.S,
				expStr)
		}
	}
}
