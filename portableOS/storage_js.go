////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// This file is only compiled for WebAssembly.

package portableOS

import (
	"gitlab.com/elixxir/wasm-utils/storage"
)

// Defines storage used by Javascript as window.localStorage.
//
//   - Specification:
//     https://html.spec.whatwg.org/multipage/webstorage.html#dom-localstorage-dev
//   - Documentation:
//     https://developer.mozilla.org/en-US/docs/Web/API/Window/localStorage
var jsStorage = storage.GetLocalStorage()
