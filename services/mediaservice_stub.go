// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build (!linux && !windows && !darwin) || android

package services

func newPlatformMediaController(svc *MediaService) (MediaController, error) {
	return noopMediaController{}, nil
}
