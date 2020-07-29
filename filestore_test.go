////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Elixxir                                                    /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package ekv

import (
	"fmt"
	"os"
	"testing"
)

type MarshalableString struct {
	S string
}

func (s *MarshalableString) Marshal() []byte {
	return []byte(s.S)
}

func (s *MarshalableString) Unmarshal(d []byte) error {
	fmt.Printf("Deserializing: %+v\n", d)
	fmt.Printf("String conversion: %s\n", string(d))
	s.S = string(d)
	fmt.Printf("New Value: %s", s.S)
	return nil
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

}
