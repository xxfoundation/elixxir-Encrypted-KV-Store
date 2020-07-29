////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Elixxir                                                    /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package ekv

import (
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

}
