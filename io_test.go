////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"fmt"
	"testing"
)

// TestModMonCntr tests all of the expected states for the Modulo Monotonic
// Counter functions.
func TestModMonCntr(t *testing.T) {
	var m1, m2 byte
	m1 = 0
	m2 = 1
	eStr := "Bad Comparison: %d > %d but returns %d"

	for i := 0; i < 10; i++ {
		g2 := compareModMonCntr(m1, m2)
		if g2 != 2 {
			t.Errorf(eStr, m2, m1, g2)
		}
		g1 := compareModMonCntr(m2, m1)
		if g1 != 1 {
			t.Errorf(eStr, m2, m1, g2)
		}

		g0 := compareModMonCntr(m1, m1)
		if g0 != 0 {
			t.Errorf("Should be invalid! %d == %d but got %d",
				m1, m1, g0)
		}
		m1 = (m1 + 1) % 3
		m2 = (m2 + 1) % 3
	}

	// Invalid comparison
	if compareModMonCntr(3, 2) != 0 {
		t.Errorf("Should be invalid!")
	}
}

func TestZeroWrite(t *testing.T) {
	key := "test"
	data := []byte{}
	err := write(key, data)
	if err == nil {
		t.Errorf("Expected error on 0 write")
	}
	expectedErr := fmt.Sprintf(errInvalidSizeContents, 0)
	if err.Error() != expectedErr {
		t.Errorf("Unexpected error: %+v", err)
	}
}
