// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

var taskbarIntegrationHook func(bool)

func SetTaskbarIntegrationHook(fn func(bool)) {
	taskbarIntegrationHook = fn
}

func setTaskbarIntegrationEnabled(enabled bool) {
	if taskbarIntegrationHook != nil {
		taskbarIntegrationHook(enabled)
	}
}
