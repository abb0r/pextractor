//go:build !windows

package main

func applyWindowIcon(interface{ ImplementsEvent() }) {}
