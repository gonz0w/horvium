// SPDX-License-Identifier: BSD-3-Clause
package inventory

// ParsePools accepts only the fixed name,guid,size,alloc,free,health projection.
// Errors return no rows; a malformed source never becomes a partial success.
func ParsePools(data []byte, limits Limits) ([]Pool, error) {
	result := make([]Pool, 0)
	names, ids := make(map[string]bool), make(map[string]bool)
	err := rows(data, limits, 6, func(f []string) error {
		if guid(f[1]) != nil || names[f[0]] || ids[f[1]] {
			return ErrMalformed
		}
		switch f[5] {
		case "ONLINE", "DEGRADED", "FAULTED", "OFFLINE", "UNAVAIL", "REMOVED", "SUSPENDED":
		default:
			return ErrMalformed
		}
		values := [3]uint64{}
		for i := range values {
			n, err := number(f[i+2])
			if err != nil {
				return err
			}
			values[i] = n
		}
		if values[1] > values[0] || values[2] > values[0] {
			return ErrMalformed
		}
		names[f[0]], ids[f[1]] = true, true
		result = append(result, Pool{Name: f[0], GUID: f[1], Size: values[0], Allocated: values[1], Free: values[2], Health: f[5]})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ParseDatasets deliberately excludes free-text mountpoints and properties.
func ParseDatasets(data []byte, limits Limits) ([]Dataset, error) {
	result := make([]Dataset, 0)
	names, ids := make(map[string]bool), make(map[string]bool)
	err := rows(data, limits, 5, func(f []string) error {
		if guid(f[1]) != nil || names[f[0]] || ids[f[1]] {
			return ErrMalformed
		}
		values := [3]uint64{}
		for i := range values {
			n, err := number(f[i+2])
			if err != nil {
				return err
			}
			values[i] = n
		}
		names[f[0]], ids[f[1]] = true, true
		result = append(result, Dataset{Name: f[0], GUID: f[1], Used: values[0], Available: values[1], Referenced: values[2]})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
