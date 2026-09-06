// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build android && cgo && !server

package main

/*
#cgo LDFLAGS: -llog
#include <android/log.h>
#include <stdlib.h>

static const char *kMelovianTag = "Melovian";

static void melovianLogError(const char *msg) {
	__android_log_write(ANDROID_LOG_ERROR, kMelovianTag, msg);
}
*/
import "C"
import "unsafe"

func mobileLog(msg string) {
	cmsg := C.CString(msg)
	defer C.free(unsafe.Pointer(cmsg))
	C.melovianLogError(cmsg)
}
