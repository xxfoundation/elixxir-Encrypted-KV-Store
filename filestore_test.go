////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/ekv/portableOS"
	"io/ioutil"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"
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
	err := portableOS.RemoveAll(".ekv_testdir")
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

	err = f.Delete("Test456")
	if err != nil {
		t.Errorf(err.Error())
	}
}

// TestFilestore_Broken tries to marshal with a broken object
func TestFilestore_Broken(t *testing.T) {
	err := portableOS.RemoveAll(".ekv_testdir_broken")
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
	err := portableOS.RemoveAll(".ekv_testdir_multiset")
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
	err := portableOS.RemoveAll(".ekv_testdir_reopen")
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

// TestFilestore_BadPass confirms using a bad password nets an error
func TestFilestore_BadPass(t *testing.T) {
	err := portableOS.RemoveAll(".ekv_testdir_badpass")
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

// TestFilestore_FDCount writes to random keys and measures that the
// number of open file descriptors is limited.
func TestFilestore_FDCount(t *testing.T) {
	// Check if we have a linux /proc/self/fd file.
	fdpath := "/proc/self/fd"
	fdStat, err := portableOS.Stat(fdpath)
	if os.IsNotExist(err) || !fdStat.IsDir() {
		t.Logf("Could not find /proc/self/fd, cannot run this test")
		return
	}

	baseDir := ".ekv_testdir_fdcount"

	getFDCount := func() int {
		files, err := ioutil.ReadDir("/proc/self/fd")
		if err != nil {
			t.Errorf(err.Error())
		}
		return len(files)
	}

	startFDCount := getFDCount()

	t.Logf("Starting File Descriptor Count: %d", startFDCount)

	err = portableOS.RemoveAll(baseDir)
	if err != nil {
		t.Errorf(err.Error())
	}

	f, err := NewFilestore(baseDir, "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	curFDCount := getFDCount()
	t.Logf("Pre-test Count: %d", curFDCount)
	startRoutinesCount := runtime.NumGoroutine()

	debug.SetGCPercent(-1)

	totalCnt := 200
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
				t.Errorf(err.Error())
			}
			s := &MarshalableString{}
			err = f.Get(keyStr, s)
			if err != nil {
				t.Errorf(err.Error())
			}
			if s.S != expStr {
				t.Errorf("Did not get what we wrote: %s != %s",
					s.S, expStr)
			}
			sharedCh <- true
			time.Sleep(100 * time.Millisecond)
			f.Delete(keyStr)
		}(f, x)
		// Kick off a read/write to the same key
		go func(f *Filestore, x int) {
			expStr := fmt.Sprintf("Hi!")
			i := &MarshalableString{
				S: expStr,
			}
			keyStrInt := "SameKey"
			err := f.SetInterface(keyStrInt, i)
			if err != nil {
				t.Errorf(err.Error())
			}
			s := &MarshalableString{}
			err = f.GetInterface(keyStrInt, s)
			if err != nil {
				t.Errorf(err.Error())
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
		select {
		case <-sharedCh:
			finishedCnt++
			curFDCount = getFDCount()
		case <-time.After(100 * time.Millisecond):
			curFDCount = getFDCount()

		}
		numRoutines := runtime.NumGoroutine() - startRoutinesCount
		t.Logf("Count at %d: %d (numProcs: %d)", finishedCnt,
			curFDCount, numRoutines)
		// Note: This number is slightly fudged.. it is based on
		// 2 files at most open per thread in the unique threads
		// and only a few threads getting past the lock on the
		// shared key threads, in practice it doesn't go above
		// ~175 or so in the corrected code when totalCnt is 200
		// whereas it always reached 400 before.
		limit := math.Max(float64(numRoutines/2), 10)
		if (curFDCount - startFDCount) > int(limit) {
			t.Errorf("Used FD Count exceeds limit: "+
				"%d > %d", curFDCount-startFDCount,
				numRoutines/2)
		}
	}

	debug.SetGCPercent(100)

}

// TestFilestore_Transaction runs 100 transactions in parallel that edit the same
// list stored to a single key. If operations are not sequential, a write will be
// written based on a read that didnt include a write that occurred after the
// writer's read, dropping the first writers data.
func TestFilestore_Transaction(t *testing.T) {
	numParalell := 100
	l := make([]int, numParalell)

	err := portableOS.RemoveAll(".ekv_testdir")
	if err != nil {
		t.Errorf(err.Error())
	}

	f, err := NewFilestore(".ekv_testdir", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	key := "test"

	if err := f.SetBytes(key, marshal(l)); err != nil {
		t.Fatalf("failed to set initial state: %+v", err)
	}

	expectedL := make([]int, numParalell)

	wg := sync.WaitGroup{}

	for i := 0; i < numParalell; i++ {
		wg.Add(1)
		go func(index int) {
			op := func(old []byte, existed bool) (data []byte, err2 error) {
				localL := unmarshal(old)
				localL[index] = index
				newData := marshal(localL)
				return newData, nil
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

func TestFilestore_Transaction_keyDoesntExist(t *testing.T) {
	numParalell := 100

	err := portableOS.RemoveAll(".ekv_testdir")
	if err != nil {
		t.Errorf(err.Error())
	}

	f, err := NewFilestore(".ekv_testdir", "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	key := "test"

	expectedL := make([]int, numParalell)

	wg := sync.WaitGroup{}

	for i := 0; i < numParalell; i++ {
		wg.Add(1)
		go func(index int) {
			op := func(old []byte, existed bool) (data []byte, err2 error) {
				var localL []int
				if !existed {
					localL = make([]int, numParalell)
				} else {
					localL = unmarshal(old)
				}
				localL[index] = index
				newData := marshal(localL)
				return newData, nil
			}
			_, _, localErr := f.Transaction(key, op)
			require.NoErrorf(t, localErr, "Transaction failed on index %s",
				index)
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

func marshal(l []int) []byte {
	b, err := json.Marshal(&l)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal: %+v", err))
	}
	return b
}

func unmarshal(b []byte) []int {
	var l []int
	err := json.Unmarshal(b, &l)
	if err != nil {
		panic(fmt.Sprintf("Failed to Unmarshal: %+v", err))
	}
	return l
}
