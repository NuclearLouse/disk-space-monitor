package datastructs

import "time"

type DiskInfo struct {
	Filesystem string
	Size       string
	Used       string
	Avail      string
	UsePrc     int
	MountedOn  string
	Server     string
	Threshold  int
	LastCheck  time.Time
}
