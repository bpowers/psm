package main

import (
	"reflect"
	"testing"
)

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
	{"    ", []string{}},
	{"abc", []string{"abc"}},
	{"abc ", []string{"abc"}},
	{"    abc", []string{"abc"}},
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
