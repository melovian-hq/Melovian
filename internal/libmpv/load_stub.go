// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build android || (!linux && !darwin && !windows)

package libmpv

func prepareCreateEnv() {}

func tryLoadLibrary() error {
	return errUnavailable
}
