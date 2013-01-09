package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	tyVmFlags      = []byte("VmFlags:")
	tyPss          = []byte("Pss:")
	tySwap         = []byte("Swap:")
	tyPrivateClean = []byte("Private_Clean:")
	tyPrivateDirty = []byte("Private_Dirty:")
)

// pidList returns a list of the process-IDs of every currently
// running process on the local system.
func pidList() ([]int, error) {
	procLs, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("ReadDir(/proc): %s", err)
	}

	pids := make([]int, 0, len(procLs))
	for _, pInfo := range procLs {
		if !isDigit(pInfo.Name()[0]) || !pInfo.IsDir() {
			continue
		}
		pidInt, err := strconv.Atoi(pInfo.Name())
		if err != nil {
			return nil, fmt.Errorf("Atoi(%s): %s", pInfo.Name(), err)
		}
		pids = append(pids, pidInt)
	}
	return pids, nil
}

// procName gets the process name for a worker.  It first checks the
// value of /proc/$PID/cmdline.  If setproctitle(3) has been called,
// it will use this.  Otherwise it uses the value of
// path.Base(/proc/$PID/exe), which has info on whether the executable
// has changed since the process was exec'ed.
func procName(pid int) (string, error) {
	p, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	// this would return an error if the PID doesn't
	// exist, or if the PID refers to a kernel thread.
	if err != nil {
		return "", nil
	}
	// cmdline is the null separated list of command line
	// arguments for the process, unless setproctitle(3)
	// has been called, in which case it is the
	argsB, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", fmt.Errorf("ReadFile(%s): %s", fmt.Sprintf("/proc/%d/cmdline", pid), err)
	}
	args := strings.Split(string(argsB), "\000")
	n := args[0]

	nTrunc := n
	if len(n) > CommMax {
		nTrunc = n[:CommMax]
	}
	if strings.HasPrefix(p, nTrunc) {
		n = path.Base(p)
	}
	return n, nil
}

// procMem returns the amount of Pss, shared, and swapped out memory
// used.  The swapped out amount refers to anonymous pages only.
func procMem(pid int) (pss, shared, heap, swap float64, err error) {
	fPath := fmt.Sprintf("/proc/%d/smaps", pid)
	f, err := os.Open(fPath)
	if err != nil {
		err = fmt.Errorf("os.Open(%s): %s", fPath, err)
		return
	}
	var priv float64
	var curr MapInfo
	r := bufio.NewReaderSize(f, pageSize)
	for {
		var l []byte
		var isPrefix bool
		l, isPrefix, err = r.ReadLine()
		// this should never happen, so take the easy way out.
		if isPrefix {
			err = fmt.Errorf("ReadLine(%s): isPrefix", fPath)
		}
		if err != nil {
			// if we've got EOF, then we're simply done
			// processing smaps.
			if err == io.EOF {
				err = nil
				break
			}
			// otherwise error out
			err = fmt.Errorf("ReadLine(%s): %s", fPath, err)
			return
		}

		if len(l) != mapDetailLen {
			if !bytes.HasPrefix(l, tyVmFlags) {
				curr = NewMapInfo(l)
			}
			continue
		}
		pieces := splitSpaces(l)
		ty := pieces[0]
		var v uint64
		if bytes.Equal(ty, tyPss) {
			v, err = ParseUint(pieces[1], 10, 64)
			if err != nil {
				err = fmt.Errorf("Atoi(%s): %s", string(pieces[1]), err)
				return
			}
			m := float64(v)
			pss += m + PssAdjust
			if curr.Name == "[heap]" {
				// we don't nead PssAdjust because
				// heap is private and anonymous.
				heap = m
			}
		} else if bytes.Equal(ty, tyPrivateClean) || bytes.Equal(ty, tyPrivateDirty) {
			v, err = ParseUint(pieces[1], 10, 64)
			if err != nil {
				err = fmt.Errorf("Atoi(%s): %s", string(pieces[1]), err)
				return
			}
			priv += float64(v)
		} else if bytes.Equal(ty, tySwap) {
			v, err = ParseUint(pieces[1], 10, 64)
			if err != nil {
				err = fmt.Errorf("Atoi(%s): %s", string(pieces[1]), err)
				return
			}
			swap += float64(v)
		}
	}
	shared = pss - priv
	return
}
