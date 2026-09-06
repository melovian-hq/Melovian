// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import "os"

func openAudioFile(path string) (*os.File, error) {
	return os.Open(path) //#nosec G304 -- path validated against library root by caller
}
