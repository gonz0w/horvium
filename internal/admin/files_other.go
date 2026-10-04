// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package admin

func LoadConfig(string) (Config, error)       { return Config{}, ErrUnsupported }
func LoadDigest(Config) ([32]byte, error)     { return [32]byte{}, ErrUnsupported }
func SetupToken(Config, bool) (string, error) { return "", ErrUnsupported }
