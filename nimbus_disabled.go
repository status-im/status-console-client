// +build !nimbus

package main

func startNimbus() {
	panic("executable needs to be built with -tags nimbus")
}
