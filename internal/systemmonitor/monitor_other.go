//go:build !linux

package systemmonitor

func readSystem(string) reading { return reading{} }
