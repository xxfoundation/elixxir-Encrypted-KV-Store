////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// TestMemstore_Smoke runs a basic read/write on the current directory.
func TestMemstore_Smoke(t *testing.T) {
	f := MakeMemstore()
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

// TestMemstore_Broken tries to marshal with a broken object.
func TestMemstore_Broken(t *testing.T) {
	f := MakeMemstore()

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

// TestMemstore_Multiset makes sure we can continuously set the object and get
// the right result each time (exercises the internal monotonic counter
// functionality).
func TestMemstore_Multiset(t *testing.T) {
	f := MakeMemstore()

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

func TestMemstore_Transaction(t *testing.T) {
	numParalell := 100
	l := make([]int, numParalell)

	f := MakeMemstore()

	key := "test"

	if err := f.SetBytes(key, marshal(l)); err != nil {
		t.Fatalf("failed to set initial state: %+v", err)
	}

	expectedL := make([]int, numParalell)

	wg := sync.WaitGroup{}

	for i := 0; i < numParalell; i++ {
		wg.Add(1)
		go func(index int) {
			op := func(old []byte, existed bool) (data []byte, deletion bool, err2 error) {
				localL := unmarshal(old)
				localL[index] = index
				newData := marshal(localL)
				return newData, false, nil
			}
			_, exist, localErr := f.Transaction(key, op)
			require.NoErrorf(t, localErr, "Transaction failed on index %s",
				index)
			require.Equal(t, exist, true, "entree did not "+
				"exist")
			wg.Done()
		}(i)
		expectedL[i] = i
	}

	wg.Wait()

	finalData, err := f.GetBytes(key)
	require.NoErrorf(t, err, "Final get errored")
	finalL := unmarshal(finalData)

	require.Equal(t, expectedL, finalL, "Writes were not sequential")
}
