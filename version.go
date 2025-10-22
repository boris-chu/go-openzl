// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

package openzl

// Version is the current version of go-openzl
const Version = "0.1.0-dev"

// OpenZLVersion returns the version of the underlying OpenZL C library
// TODO: Implement this once CGO bindings are in place
func OpenZLVersion() string {
	return "unknown"
}
