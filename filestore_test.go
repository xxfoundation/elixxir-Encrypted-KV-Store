////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/ekv/portable"
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
	defer func() {
		if err := portable.UsePosix().RemoveAll(".ekv_testdir"); err != nil {
			t.Error(err)
		}
	}()

	f, err := NewFilestore(".ekv_testdir", "Hello, World!")
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

// TestFilestore_Broken tries to marshal with a broken object
func TestFilestore_Broken(t *testing.T) {
	defer func() {
		if err := portable.UsePosix().RemoveAll(".ekv_testdir_broken"); err != nil {
			t.Error(err)
		}
	}()

	f, err := NewFilestore(".ekv_testdir_broken", "Hello, World 22!")
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
		t.Errorf("Unmarshal succeded!")
	}
}

// TestFilestore_Multiset makes sure we can continuously set the object and get
// the right result each time (exercises the internal monotonic counter
// functionality)
func TestFilestore_Multiset(t *testing.T) {
	defer func() {
		if err := portable.UsePosix().RemoveAll(".ekv_testdir_multiset"); err != nil {
			t.Error(err)
		}
	}()

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

// TestFilestore_Reopen verifies we can recreate/reopen the store and get the
// data we stored back out.
func TestFilestore_Reopen(t *testing.T) {
	defer func() {
		if err := portable.UsePosix().RemoveAll(".ekv_testdir_reopen"); err != nil {
			t.Error(err)
		}
	}()

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
		t.Error(err)
	}
	// Now test set/get Interface
	err = f.SetInterface("Test456", i)
	if err != nil {
		t.Error(err)
	}

	for x := 0; x < 20; x++ {
		f, err = NewFilestore(".ekv_testdir_reopen", "Hello, World!")
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

// TestFilestore_BadPass confirms using a bad password nets an error
func TestFilestore_BadPass(t *testing.T) {
	defer func() {
		if err := portable.UsePosix().RemoveAll(".ekv_testdir_badpass"); err != nil {
			t.Error(err)
		}
	}()

	_, err := NewFilestore(".ekv_testdir_badpass", "Hello, World!")
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
	// Check if we can count file descriptors on this platform
	startFDCount, err := getFDCount()
	if err != nil {
		t.Skipf("Cannot count file descriptors on this platform: %v", err)
		return
	}

	baseDir := ".ekv_testdir_fdcount"

	t.Logf("Starting File Descriptor Count: %d", startFDCount)

	err = portable.UsePosix().RemoveAll(baseDir)
	if err != nil {
		t.Error(err)
	}

	f, err := NewFilestore(baseDir, "Hello, World!")
	if err != nil {
		t.Errorf("%+v", err)
	}

	curFDCount, err := getFDCount()
	if err != nil {
		t.Errorf("Failed to get FD count: %v", err)
	}
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
			time.Sleep(100 * time.Millisecond)
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
	maxDeltaFD := 0
	for finishedCnt < totalCnt*2 {
		select {
		case <-sharedCh:
			finishedCnt++
			curFDCount, _ = getFDCount()
		case <-time.After(200 * time.Millisecond):
			// Only check FD count periodically if we still have active goroutines
			numRoutines := runtime.NumGoroutine() - startRoutinesCount
			if numRoutines == 0 && finishedCnt == totalCnt*2 {
				// All done, exit the loop
				break
			}
			curFDCount, _ = getFDCount()
		}

		numRoutines := runtime.NumGoroutine() - startRoutinesCount
		deltaFD := curFDCount - startFDCount
		if deltaFD > maxDeltaFD {
			maxDeltaFD = deltaFD
		}

		// Only log when there's activity or we're sampling
		if finishedCnt%10 == 0 || numRoutines > 0 {
			t.Logf("Progress %d/%d: +%d FDs (active goroutines: %d, peak: +%d FDs)",
				finishedCnt, totalCnt*2, deltaFD, numRoutines, maxDeltaFD)
		}

		// The goal is to ensure FDs don't leak, not to have perfect concurrency tuning.
		// We check that FD count doesn't grow unbounded (which would indicate leaks).
		// If leaking, we'd see FD count approach totalCnt*2 (400+) and stay there.
		// Instead, we see FDs spike during concurrency then return to baseline.
		//
		// The limit is intentionally generous because:
		// - We can't perfectly predict concurrent behavior
		// - macOS proc_pidinfo may count differently than Linux /proc/self/fd
		// - The real validation is in the final check (FDs return to baseline)
		const maxAllowedFDs = 200
		if deltaFD > maxAllowedFDs {
			t.Errorf("FD count suggests possible leak: +%d FDs (expected < %d)",
				deltaFD, maxAllowedFDs)
		}
	}

	// Final check: FDs should return to baseline
	// Give goroutines time to finish and GC to clean up
	for i := 0; i < 3; i++ {
		runtime.GC()
		time.Sleep(50 * time.Millisecond)
	}

	finalFDCount, _ := getFDCount()
	finalDelta := finalFDCount - startFDCount
	t.Logf("Final: +%d FDs (peak was +%d FDs)", finalDelta, maxDeltaFD)

	if finalDelta > 5 {
		t.Errorf("File descriptors not properly closed: %d FDs still open after test", finalDelta)
	}

	debug.SetGCPercent(100)

}
