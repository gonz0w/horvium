// SPDX-License-Identifier: BSD-3-Clause
package main

// An EOF exit is distinct from forced cancellation or a successful release.
func witnessConnectorWaitMatches(action string, exited bool, code, signal int) bool {
	switch action {
	case "release":
		return exited && code == 0 && signal == 0
	case "channel-eof":
		return exited && code == 1 && signal == 0
	case "sigterm":
		return !exited && code == 0 && signal == 15
	case "sigkill":
		return !exited && code == 0 && signal == 9
	}
	return false
}

type witnessConnectorWaitRecord struct {
	Present, Exited  bool
	ExitCode, Signal int
	Status           uint32
}

func witnessConnectorWaitRepresentation(present, exited bool, code, signal int, status uint32) witnessConnectorWaitRecord {
	if !present || (exited && code < 0) || signal < 0 {
		present = false
		code = 0
		signal = 0
	}
	if !exited {
		code = 0
	}
	return witnessConnectorWaitRecord{present, exited, code, signal, status}
}
