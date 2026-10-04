//go:build !freebsd && !darwin && !linux

// SPDX-License-Identifier: BSD-3-Clause
package main

import "os"

func ownedInput(os.FileInfo) bool { return false }

func privatePipe(os.FileInfo) bool { return false }
