# Encrypted KV Store

[![pipeline status](https://gitlab.com/elixxir/ekv/badges/master/pipeline.svg)](https://gitlab.com/elixxir/ekv/commits/master)
[![coverage report](https://gitlab.com/elixxir/ekv/badges/master/coverage.svg)](https://gitlab.com/elixxir/ekv/commits/master)


EKV is a directory and file-based encrypted key value storage library
with metadata protection written in golang. It is intended for use in
mobile and desktop applications where one may want to transfer
protected files to a new device while protecting the nature of the
information of what is stored in addition to the contents.

Features:
1. Both the key and the contents behind the key are protected on disk.
2. A best-effort approach is used to store and flush changes to disk.
3. Thread-safety within the same EKV instance.

EKV is not a secured memory enclave. Data is protected when stored to
disk, not RAM. Please see
[Memguard](https://github.com/awnumar/memguard) and similar projects
if that is what you need.

EKV requires a cryptographically secure random number generator. We
recommend Elixxir's fastRNG.

EKV is released under the simplified BSD License.

## Known Limitations and Roadmap

EKV has several known limitations at this time:

1. The code is currently in beta and has not been audited.
2. The password to open and close the store is a string that can be
dumped from memory. We would like to improve that by storing the
password in a secured memory enclave (e.g.,
[Memguard](https://github.com/awnumar/memguard)).
3. EKV protects keys and contents, it doesn't protect the size of
those files or the number of unique keys being stored in the
database. We would like to include controls for EKV users to hide that
information by setting a block size for files and adding a number of
fake files to the directory.
4. Users are currently limited to the number of files the operating
system can support in a single directory.
5. The underlying file system must support hex encoded 256 bit file
names.

## General Usage

EKV implements the following interface:


```
type KeyValue interface {
	Set(key string, objectToStore Marshaler) error
	Get(key string, loadIntoThisObject Unmarshaler) error
	Delete(key string) error
	SetInterface(key string, objectToSTore interface{}) error
	GetInterface(key string, v interface{}) error
}
```

EKV works with any object that implements the following functions:

1. Marhsaler: `func Marshal() []byte`
2. Unmarshaler: `func Unmarshal ([]byte) error`

For example, we can make a "MarshalableString" type:

```
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
```

To load and store to the EKV with this type:

```
import (
	...
	"crypto/rand"
	"gitlab.com/elixxir/ekv"
)

func main() {
	kvstore, err := ekv.NewFilestoreWithNonceGenerator("somedirectory",
		"Some Password", rand.Reader)
	if err != nil {
		// Print/handle could not create or open error ...
	}

	i := &MarshalableString{
		S: "TheValue",
	}
	err = f.Set("SomeKey", i)
	if err != nil {
		// Print/handle could not write error ...
	}

	s := &MarshalableString{}
	err = f.Get("SomeKey", s)
	if err != nil {
		// Print/handle could not read error
	}
	if s.S == "Hi" {
		// Always true
	}
}
```

### Generic Interfaces (JSON Encoding)

You can also leverage the default JSON Marshalling using
`GetInterface` and `SetInterface` as follows:

```
	err = f.SetInterface("SomeKey", i)
	if err != nil {
		// write error
	}

	s = &MarshalableString{}
	err = f.GetInterface("SomeKey", s)
	if err != nil {
		// read error
	}
	if s.S == "Hi" {
		// Always true
	}
```

### Deleting Data

To delete, use `Delete`, which will also remove the file corresponding
to the key:

```
	err = f.Delete("SomeKey")
	if err != nil {
		// Could not delete
	}
```

### Detecting if a key exists:

To detect if a key exists you can use the `Exists` function on the
error returned by `Get` and `GetInterface`:

```
	err = f.GetInterface("SomeKey", s)
	if !ekv.Exists(err) {
		// Does not exist...
	}
```

# Cryptographic Primitives

All cryptographic code is located in `crypto.go`.

To create keys, EKV uses the construct:

* `H(H(password)||H(keyname))`

The `keyname` is the name of the key and `password` is the password or
passphrase used to generate the key. EKV uses the 256bit blake2b hash.

Code:


```
func hashStringWithPassword(data, password string) []byte {
	dHash := blake2b.Sum256([]byte(data))
	pHash := blake2b.Sum256([]byte(password))
	s := append(pHash[:], dHash[:]...)
	h := blake2b.Sum256(s)
	return h[:]
}
```


To encrypt files, EKV uses ChaCha20Poly1305 with a randomly generated
nonce. The cryptographically secure pseudo-random number generator
must be provided by the user:


```
func initChaCha20Poly1305(password string) cipher.AEAD {
	pwHash := blake2b.Sum256([]byte(password))
	chaCipher, err := chacha20poly1305.NewX(pwHash[:])
	if err != nil {
		panic(fmt.Sprintf("Could not init XChaCha20Poly1305 mode: %s",
			err.Error()))
	}
	return chaCipher
}

func encrypt(data []byte, password string, csprng io.Reader) []byte {
	chaCipher := initChaCha20Poly1305(password)
	nonce := make([]byte, chaCipher.NonceSize())
	if _, err := io.ReadFull(csprng, nonce); err != nil {
		panic(fmt.Sprintf("Could not generate nonce: %s", err.Error()))
	}
	ciphertext := chaCipher.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func decrypt(data []byte, password string) ([]byte, error) {
	chaCipher := initChaCha20Poly1305(password)
	nonceLen := chaCipher.NonceSize()
	nonce, ciphertext := data[:nonceLen], data[nonceLen:]
	plaintext, err := chaCipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot decrypt with password!")
	}
	return plaintext, nil
}
```
