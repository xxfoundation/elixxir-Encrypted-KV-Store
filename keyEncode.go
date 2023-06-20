////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// This file is compiled for all architectures except WebAssembly.
//go:build !js || !wasm

package ekv

import (
	"encoding/hex"
)

// encodeKey encodes a Filestore key using hex encoding.
func encodeKey(key []byte) string {
	return hex.EncodeToString(key)
}
