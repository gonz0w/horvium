// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "context"

// executeNative accepts only projections supplied by readNative. Tests may use
// their own reexecuted binary; requests never supply an executable or argument.
func executeNative(ctx context.Context, path string, args []string, limit int) (readOutput, error) {
	return execute(ctx, path, args, limit)
}
