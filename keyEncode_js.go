////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"github.com/Max-Sum/base32768"
)

// encodeKey encodes a Filestore key using base 32768 encoding.
func encodeKey(key []byte) string {
	return base32768.SafeEncoding.EncodeToString(key)
}
