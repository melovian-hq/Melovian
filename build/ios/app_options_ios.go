// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build ios

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS adjusts the application options for iOS
func modifyOptionsForIOS(opts *application.Options) {
	opts.DisableDefaultSignalHandler = true
	opts.IOS.EnableAutoplayWithoutUserAction = true
	opts.IOS.EnableInlineMediaPlayback = true
	opts.IOS.BackgroundColour = application.NewRGB(9, 9, 9)
}
