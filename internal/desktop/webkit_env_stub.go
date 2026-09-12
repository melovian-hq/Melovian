// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package desktop

func ApplyWebKitStabilityEnv(_ string) map[string]string {
	return nil
}

func AppliedWebKitEnv() map[string]string {
	return map[string]string{}
}

func GraphicsEnvironmentInfo(_ string) GraphicsEnvironment {
	return GraphicsEnvironment{Supported: false}
}

func DisableWebKitSandboxIfLandlocked(_ bool) {}
