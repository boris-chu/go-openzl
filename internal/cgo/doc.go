// Copyright (c) 2025 Boris Chu and contributors
// SPDX-License-Identifier: BSD-3-Clause

// Package cgo provides low-level CGO bindings to the OpenZL C library.
//
// This package is internal and should not be used directly. Use the openzl
// package instead, which provides a safe, idiomatic Go interface.
//
// The bindings in this package are thin wrappers around the OpenZL C API,
// handling memory management, error translation, and type conversions.
package cgo
