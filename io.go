///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ekv

// io.go provides generic, internal read and write functions which try to
// guarantee writes are committed to disk. Our general strategy is to
// use two files, writing over the "oldest" and reading the "newest". Additional
// precautions are taken to ensure the file is written to the directory and that
// it is flushed to disk every time.

// The "oldest" and "newest" are determined using a modular monotonic counter
// (ModMonCntr), which "always increases" in a modular monotonic sense. In other
// words: 0 < 1 < 2 < 0 and so on forever.

// Because ekv is meant to load/store in-memory objects, reads and writes are
// done on the entire file and never incrementally.

// NOTE: We assume calls to this library are synchronized and that the data is
// not modified by external programs. It's possible to break things if an
// external program modifies the first byte or we don't enforce synchronized
// calls to these functions using a mutex for the same filename.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/ekv/portableOS"
	"golang.org/x/crypto/blake2b"
	"io"
	"os"
	"path/filepath"
)

const (
	errModMonCntrShortRead  = "ModMonCntr read: %d bytes read, not 1"
	errModMonCntrShortWrite = "ModMonCntr write: wrote %d bytes, not 1"
	errModMonCntrInvalidVal = "ModMonCntr Invalid Values: %d, %d"
	errNewestFile           = "Invalid read finding newest file: %s"
	errShortRead            = "Short file read %s: got %d, expected %d"
	errShortWrite           = "Short file write %s: got %d, expected %d"
	errInvalidSizeContents  = "Invalid contents size: %d"
	errChecksum             = "Invalid Checksum %s: Actual(%X) != Expected(%X)"
	errCannotRead           = "Did not read the same data that was written!"
	errIsDir                = "File path is a directory: %s"
	errInvalidFile          = "Invalid file"
	modMonCntrSize          = 1
)

// getPaths returns "path.1" and "path.2"
func getPaths(path string) (string, string) {
	f1 := fmt.Sprintf("%s.1", path)
	f2 := fmt.Sprintf("%s.2", path)
	return f1, f2
}

// compareModMonCntr returns 1 if t1 is newer, 2 if t2 is newer, and 0 if
// there is an error. newer is defined as the second of 3 cases:
// (0 < 1), (1 < 2), (2 < 0). Anything else is an error
func compareModMonCntr(t1, t2 byte) byte {
	// NOTE: Yes, the following could be cleverer -- don't "improve" it.
	// t1 cases, 1 > 0, 2 > 1 and 0 > 2
	if (t1 == 1 && t2 == 0) ||
		(t1 == 2 && t2 == 1) ||
		(t1 == 0 && t2 == 2) {
		return 1
	}

	// t2 cases, 0 < 1, 1 < 2, and 2 < 0
	if (t1 == 0 && t2 == 1) ||
		(t1 == 1 && t2 == 2) ||
		(t1 == 2 && t2 == 0) {
		return 2
	}

	// everything else is an error
	return 0
}

// getFileOrder returns the newest and oldest files using the modular monotic
// counter inside them. If either fails to read, the successful file is returned
// if both fail to read, or return invalid results, return an error.
func getFileOrder(path1, path2 string) (portableOS.File, portableOS.File, error) {
	// default to invalid values. The only valid modulo monotonic counter
	// values are 0, 1, and 2.
	t1 := byte(3)
	t2 := byte(3)

	buf := make([]byte, 1)

	// Try to open and read file1
	file1, err1 := portableOS.Open(path1)
	if err1 == nil {
		buf[0] = 3
		_, err1 = file1.ReadAt(buf, 0)
		t1 = buf[0]
	}
	// Try to open and read file2
	file2, err2 := portableOS.Open(path2)
	if err2 == nil {
		buf[0] = 3
		_, err2 = file2.ReadAt(buf, 0)
		t2 = buf[0]
	}

	// If both files don't exist, return that
	if os.IsNotExist(err1) && os.IsNotExist(err2) {
		return nil, nil, err1
	}

	// Otherwise return composite error
	if err1 != nil && err2 != nil {
		return nil, nil, errors.Errorf(errNewestFile+", %s", err1,
			err2)
	}

	// Return file 2 or file 1 if one of them did not error out
	if err1 != nil {
		return file2, nil, nil
	}
	if err2 != nil {
		return file1, nil, nil
	}

	// Otherwise compare the modulo monotonic counter and return the result
	cmp := compareModMonCntr(t1, t2)
	if cmp == 1 {
		return file1, file2, nil
	}
	if cmp == 2 {
		return file2, file1, nil
	}

	return nil, nil, errors.Errorf(errModMonCntrInvalidVal, t1, t2)
}

// readContents of a file, checking the checksum and returning the data.
// this function assumes the file read header is at the beginning of the content
// block
func readContents(f portableOS.File) ([]byte, error) {
	// Read the contents size
	sizeBytes := make([]byte, 4)
	_, _ = f.Seek(1, 0)
	cnt, err := f.Read(sizeBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error reading size")
	}
	if cnt != len(sizeBytes) {
		return nil, errors.Errorf(errShortRead, f.Name(),
			cnt, len(sizeBytes))
	}
	size := int(binary.LittleEndian.Uint32(sizeBytes))
	if size <= 0 {
		errors.Errorf(errInvalidSizeContents, size)
	}

	// Read the contents
	contents := make([]byte, size)
	cnt, err = f.Read(contents)
	if err != nil {
		return nil, errors.Wrap(err, "error reading contents")
	}
	if cnt != size {
		return nil, errors.Errorf(errShortRead, f.Name(), cnt, size)
	}

	// Read checksum
	checksumInFile := make([]byte, blake2b.Size256)
	cnt, err = f.Read(checksumInFile)
	if err != nil {
		return nil, errors.Wrap(err, "error reading checksum")
	}
	if cnt != blake2b.Size256 {
		return nil, errors.Errorf(errShortRead, f.Name(),
			cnt, blake2b.Size256)
	}

	actualChecksum := blake2b.Sum256(contents)
	if !bytes.Equal(checksumInFile, actualChecksum[:]) {
		return nil, errors.Errorf(errChecksum, f.Name(), actualChecksum,
			checksumInFile)
	}

	return contents, nil
}

// createFile creates the file, flushes the directory then returns an open,
// writable file handle
func createFile(path string) (portableOS.File, error) {
	// Create file if is it is a "does not exist error"
	f, err := portableOS.Create(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	f.Sync()
	f.Close()

	// Open directory and flush it
	dirname := filepath.Dir(path)
	d, err := portableOS.Open(dirname)
	d.Sync()
	d.Close()

	return portableOS.Create(path)
}

// deleteFile overwrites a files contents with random data and then deletes
// the file
func deleteFile(path string, csprng io.Reader) error {
	info, err := portableOS.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	// If the file exists, attempt to delete
	if err != nil {
		return err
	}

	buf := make([]byte, info.Size())
	if _, err = io.ReadFull(csprng, buf); err != nil {
		return err
	}
	f, err := portableOS.Create(path)
	if err != nil {
		return err
	}
	_, err = f.Write(buf)
	if err != nil {
		return err
	}
	f.Close()
	f.Sync()
	err = portableOS.Remove(path)
	return err
}

// deleteFiles deletes both files and then flushes the directory
func deleteFiles(path string, csprng io.Reader) error {
	// Create file if is it is a "does not exist error"
	var fns [2]string
	fns[0], fns[1] = getPaths(path)

	// Delete both paths if they exist
	for i := 0; i < 2; i++ {
		err := deleteFile(fns[i], csprng)
		// Return errors from removal OR stat check
		if err != nil {
			return err
		}
	}

	// Open directory and flush it
	dirname := filepath.Dir(path)
	d, err := portableOS.Open(dirname)
	d.Sync()
	d.Close()

	return err
}

// write to the file and verify the data can be read
func write(path string, data []byte) error {
	if len(data) == 0 {
		return errors.New(fmt.Sprintf(errInvalidSizeContents, 0))
	}
	// First, check if either file can be read. Then write to the other one
	path1, path2 := getPaths(path)
	newest, oldest, err := getFileOrder(path1, path2)
	if newest != nil {
		defer newest.Close()
	}
	if oldest != nil {
		defer oldest.Close()
	}

	filesToRead := []portableOS.File{newest, oldest}
	modMonCntr := byte(2) // (2+1)%3 defaults to 0 when we can't read it
	filePathThatWasRead := ""
	for i := 0; i < len(filesToRead); i++ {
		if filesToRead[i] == nil {
			continue
		}
		buf := make([]byte, 1)
		buf[0] = 3
		cnt, _ := filesToRead[i].ReadAt(buf, 0)
		_, err := readContents(filesToRead[i])

		if cnt == 1 && err == nil {
			modMonCntr = buf[0]
			filePathThatWasRead = filesToRead[i].Name()
			break
		}
	}

	// Set the file to write, based on which file was read, if any
	var fileToWrite portableOS.File
	var filePathToWrite string
	if filePathThatWasRead == "" || filePathThatWasRead == path2 {
		filePathToWrite = path1
	} else {
		filePathToWrite = path2
	}

	// Write the counter and contents of the file
	modMonCntr = (modMonCntr + 1) % 3
	// modMonCntrSize + 4 bytes to represent data len, len of data,
	// and 256 bit (32 byte) hash size
	contents := make([]byte, 1+4+len(data)+32)
	contents[0] = modMonCntr

	// Copy in the size
	size := len(data)
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(size))
	// Bytes 1:4 are the size
	copy(contents[1:4], sizeBytes)

	// Bytes 5 -> 5 + len(data) - 1 are the contents
	contentStart := 5
	contentEnd := contentStart + size
	copy(contents[contentStart:contentEnd], data)

	// Checksum at the end
	checksum := blake2b.Sum256(data)
	csumStart := contentEnd
	csumEnd := csumStart + blake2b.Size256
	copy(contents[csumStart:csumEnd], checksum[:])

	fileToWrite, err = createFile(filePathToWrite)
	// Error out if we failed to create
	if err != nil {
		return err
	}

	n, err := fileToWrite.Write(contents)
	if err != nil {
		fileToWrite.Close()
		return err
	}
	if n != len(contents) {
		fileToWrite.Close()
		return errors.Errorf(errShortWrite, filePathToWrite,
			n, len(contents))
	}

	fileToWrite.Sync()
	fileToWrite.Close()

	// Check that what we wrote is equal to what we have
	fileToWrite, err = portableOS.Open(filePathToWrite)
	if err != nil {
		return err
	}
	contentsToCheck, err := readContents(fileToWrite)
	fileToWrite.Close()
	if err != nil {
		return err
	}

	if !bytes.Equal(data, contentsToCheck) {
		return errors.Errorf(errCannotRead)
	}

	return nil
}

// read returns the contents of the newest file for which it
// can read all elements and validate the internal checksum
func read(path string) ([]byte, error) {
	// Open the newest first, note we only return this error if
	// both returned file objects are bad (e.g., if neither file exists or
	// the first byte of both files cannot be read)
	path1, path2 := getPaths(path)
	newest, oldest, err := getFileOrder(path1, path2)
	if newest != nil {
		defer newest.Close()
	}
	if oldest != nil {
		defer oldest.Close()
	}

	// Return the first file we can read the contents and validate a
	// checksum, or an error
	filesToRead := []portableOS.File{newest, oldest}
	for i := 0; i < len(filesToRead); i++ {
		if filesToRead[i] == nil {
			continue
		}
		contents, err := readContents(filesToRead[i])
		if err != nil {
			continue
		}
		if len(contents) != 0 {
			return contents, nil
		}
	}

	// Read and return the contents
	return nil, err
}
