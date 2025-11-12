////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// This file is only compiled for WebAssembly.
//go:build js && wasm

package portable

// UsePosix is not available in WebAssembly environments. Use UseKeyValue
// instead with a JavaScript key-value store (e.g., localStorage, IndexedDB).
func UsePosix() Storage {
	panic("UsePosix is not available in WebAssembly; use UseKeyValue with a JavaScript key-value store instead")
}
