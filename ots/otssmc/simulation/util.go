package main

import (
	"syscall"

	"gopkg.in/dedis/onet.v1/log"
)

func iiToF(sec int64, usec int64) float64 {
	return float64(sec) + float64(usec)/1000000.0
}

// Returns the system and the user CPU time used by the current process so far.
func GetRTime() (tSys, tUsr float64) {
	rusage := &syscall.Rusage{}
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, rusage); err != nil {
		log.Error("Couldn't get rusage time:", err)
		return -1, -1
	}
	s, u := rusage.Stime, rusage.Utime
	return iiToF(int64(s.Sec), int64(s.Usec)), iiToF(int64(u.Sec), int64(u.Usec))
}

func GetDiffRTime(tSys, tUsr float64) (tDiffSys, tDiffUsr float64) {
	nowSys, nowUsr := GetRTime()
	return nowSys - tSys, nowUsr - tUsr
}
