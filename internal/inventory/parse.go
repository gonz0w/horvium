// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"bytes"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func checkInput(data []byte, limits Limits) error {
	if err := limits.Validate(); err != nil {
		return err
	}
	if len(data) > limits.MaxBytes {
		return ErrLimit
	}
	if !utf8.Valid(data) {
		return ErrMalformed
	}
	return nil
}

func safeText(value string, limits Limits) error {
	if len(value) > limits.MaxFieldBytes {
		return ErrLimit
	}
	if value == "" || !utf8.ValidString(value) {
		return ErrMalformed
	}
	for _, c := range value {
		if unicode.IsControl(c) {
			return ErrMalformed
		}
	}
	return nil
}

func number(value string) (uint64, error) {
	if value == "" {
		return 0, ErrMalformed
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, ErrMalformed
		}
	}
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, ErrMalformed
	}
	return n, nil
}

func guid(value string) error {
	n, err := number(value)
	if err != nil || n == 0 || strconv.FormatUint(n, 10) != value {
		return ErrMalformed
	}
	return nil
}

func rows(data []byte, limits Limits, count int, visit func([]string) error) error {
	if err := checkInput(data, limits); err != nil {
		return err
	}
	for records := 0; len(data) != 0; records++ {
		if records >= limits.MaxRecords {
			return ErrLimit
		}
		end := bytes.IndexByte(data, '\n')
		if end == -1 {
			return ErrMalformed
		}
		// Reject a delimiter flood before Split allocates a field slice.
		if bytes.Count(data[:end], []byte{'\t'}) != count-1 {
			return ErrMalformed
		}
		fields := strings.Split(string(data[:end]), "\t")
		if len(fields) != count {
			return ErrMalformed
		}
		for _, field := range fields {
			if err := safeText(field, limits); err != nil {
				return err
			}
		}
		if err := visit(fields); err != nil {
			return err
		}
		data = data[end+1:]
	}
	return nil
}
