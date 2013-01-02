package main

import (
	"reflect"
	"strings"
	"testing"
)

const benchString = `7fff70e93000-7fff70eb5000 rw-p 00000000 00:00 0                          [stack]
Size:                140 kB
Rss:                  12 kB
Pss:                  12 kB
Shared_Clean:          0 kB
Shared_Dirty:          0 kB
Private_Clean:         0 kB
Private_Dirty:        12 kB
Referenced:           12 kB
Anonymous:            12 kB
AnonHugePages:         0 kB
Swap:                  0 kB
KernelPageSize:        4 kB
MMUPageSize:           4 kB
Locked:                0 kB
VmFlags: rd wr mr mw me gd ac 
7fff70fff000-7fff71000000 r-xp 00000000 00:00 0                          [vdso]
Size:                  4 kB
Rss:                   4 kB
Pss:                   0 kB
Shared_Clean:          4 kB
Shared_Dirty:          0 kB
Private_Clean:         0 kB
Private_Dirty:         0 kB
Referenced:            4 kB
Anonymous:             0 kB
AnonHugePages:         0 kB
Swap:                  0 kB
KernelPageSize:        4 kB
MMUPageSize:           4 kB
Locked:                0 kB
VmFlags: rd ex mr mw me de 
ffffffffff600000-ffffffffff601000 r-xp 00000000 00:00 0                  [vsyscall]
Size:                  4 kB
Rss:                   0 kB
Pss:                   0 kB
Shared_Clean:          0 kB
Shared_Dirty:          0 kB
Private_Clean:         0 kB
Private_Dirty:         0 kB
Referenced:            0 kB
Anonymous:             0 kB
AnonHugePages:         0 kB
Swap:                  0 kB
KernelPageSize:        4 kB
MMUPageSize:           4 kB
Locked:                0 kB
VmFlags: rd ex 
`

var (
	benchStrLines = strings.Split(benchString, "\n")
	benchLines    = sliceByteArr(benchStrLines)
	usedResult    [][]byte
)

func sliceByteArr(b []string) [][]byte {
	res := make([][]byte, len(b))
	for i, bs := range b {
		res[i] = []byte(bs)
	}
	return res
}

func stringArr(b [][]byte) []string {
	res := make([]string, len(b))
	for i, bs := range b {
		res[i] = string(bs)
	}
	return res
}

var splitSpacesData = [...]struct {
	orig  string
	split []string
}{
	{"", []string{}},
	{" ", []string{}},
	{"    ", []string{}},
	{"abc", []string{"abc"}},
	{"abc ", []string{"abc"}},
	{"    abc ", []string{"abc"}},
	{"abc 123", []string{"abc", "123"}},
	{"abc 123    ", []string{"abc", "123"}},
	{"abc    123", []string{"abc", "123"}},
	{"   abc    123", []string{"abc", "123"}},
	{"   abc    123 def", []string{"abc", "123", "def"}},
}

func TestSplitSpaces(t *testing.T) {
	for _, pair := range splitSpacesData {
		origB := []byte(pair.orig)
		ss := stringArr(splitSpaces(origB))
		if !reflect.DeepEqual(ss, pair.split) {
			t.Fatalf("expected equal:\n    orig: %#v\n    ref:  %#v\n",
				pair.split, ss)
		}
	}
}

func BenchmarkSplitSpaces(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, l := range benchLines {
			usedResult = splitSpaces(l)
		}
	}
}
