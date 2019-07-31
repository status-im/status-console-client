// +build nimbus

package main

import (
	statusnim "github.com/status-im/status-nim"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func startNimbus() {
	statusnim.Start()
	statusnim.ListenAndPost()
}
