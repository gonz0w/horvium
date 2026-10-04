// SPDX-License-Identifier: BSD-3-Clause
package helper

import "github.com/gonz0w/horvium/internal/inventory"

func validateSnapshot(s inventory.Snapshot) error {
	if inventory.ValidateSnapshot(s) != nil {
		return ErrProtocol
	}
	return nil
}
