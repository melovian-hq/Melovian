// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build android && !server

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func modifyOptionsForIOS(opts *application.Options) {}
