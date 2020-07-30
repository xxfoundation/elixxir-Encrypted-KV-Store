////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Elixxir                                                    /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package ekv

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"testing"
)

// This is a simple marshalable object
type MarshalableString struct {
	S string
}

func (s *MarshalableString) Marshal() []byte {
	return []byte(s.S)
}

func (s *MarshalableString) Unmarshal(d []byte) error {
	s.S = string(d)
	return nil
}

// This breaks every time you try to unmarshal
type BrokenMarshalable struct {
	S string
}

func (s *BrokenMarshalable) Marshal() []byte {
	return []byte(s.S)
}

func (s *BrokenMarshalable) Unmarshal(d []byte) error {
	return errors.New("can't unmarshal")
}

// TestFilestore_Smoke runs a basic read/write on the current directory
func TestFilestore_Smoke(t *testing.T) {
	err := os.RemoveAll(".ekv_testdir")
	if err != nil {
		t.Errorf(err.Error())
	}

	f, err := NewFilestore(".ekv_testdir", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	i := &MarshalableString{
		S: "Hi",
	}
	err = f.Set("TestMe123", i)
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
func TestFilestore_Broken(t *testing.T) {
	err := os.RemoveAll(".ekv_testdir_broken")
	if err != nil {
		t.Errorf(err.Error())
	}

	f, err := NewFilestore(".ekv_testdir_broken", "Hello, World 22!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	i := &BrokenMarshalable{
		S: "Hi",
	}
	err = f.Set("TestMe123", i)
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
func TestFilestore_Multiset(t *testing.T) {
	err := os.RemoveAll(".ekv_testdir_multiset")
	if err != nil {
		t.Errorf(err.Error())
	}

	f, err := NewFilestore(".ekv_testdir_multiset", "Hello, World!")
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

// TestFilestore_Reopen verifies we can recreate/reopen the store and get the
// data we stored back out.
func TestFilestore_Reopen(t *testing.T) {
	err := os.RemoveAll(".ekv_testdir_reopen")
	if err != nil {
		t.Errorf(err.Error())
	}

	f, err := NewFilestore(".ekv_testdir_reopen", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	expStr := "Hi"

	i := &MarshalableString{
		S: expStr,
	}
	err = f.Set("TestMe123", i)
	if err != nil {
		t.Errorf(err.Error())
	}
	// Now test set/get Interface
	err = f.SetInterface("Test456", i)
	if err != nil {
		t.Errorf(err.Error())
	}

	for x := 0; x < 20; x++ {
		f, err = NewFilestore(".ekv_testdir_reopen", "Hello, World!")
		if err != nil {
			t.Errorf("%+v", err)
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
		s = &MarshalableString{}
		err = f.GetInterface("Test456", s)
		if err != nil {
			t.Errorf(err.Error())
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
			t.Errorf(err.Error())
		}
		// Now test set/get Interface
		err = f.SetInterface("Test456", i)
		if err != nil {
			t.Errorf(err.Error())
		}
	}
}

// TestFilestore_Reopen verifies we can recreate/reopen the store and get the
// data we stored back out.
func TestFilestore_BadPass(t *testing.T) {
	err := os.RemoveAll(".ekv_testdir_badpass")
	if err != nil {
		t.Errorf(err.Error())
	}

	_, err = NewFilestore(".ekv_testdir_badpass", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	_, err = NewFilestore(".ekv_testdir_badpass", "badpassword")
	if err == nil {
		t.Errorf("Opened with bad password!")
	}

}
