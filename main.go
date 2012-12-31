package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	CmdDisplayMax = 32
	// max length of /proc/$PID/comm
	CommMax = 16
	// from ps_mem - average error due to truncation in the kernel
	// pss calculations
	PssAdjust = .5
	pageSize = 4096
)

// store info about a command (group of processes), similar to how
// ps_mem works.
type CmdMemInfo struct {
	PIDs    []int
	Name    string
	Pss     float64
	Shared  int64
	Private int64
	Swapped int64
}

type MapInfo struct {
	Inode int64
	Name  string
}

// mapLine is a line from /proc/$PID/maps, or one of the same header
// lines from smaps.
func NewMapInfo(mapLine []byte) MapInfo {
	var mi MapInfo
	var err error
	pieces := splitSpaces(mapLine)
	if len(pieces) == 6 {
		mi.Name = string(pieces[5])
	}
	mi.Inode, err = strconv.ParseInt(string(pieces[4]), 10, 64)
	if err != nil {
		panic(fmt.Sprintf("NewMapInfo: Atoi(%s): %s (%s)",
			string(pieces[4]), err, string(mapLine)))
	}
	return mi
}

func (mi MapInfo) IsAnon() bool {
	return mi.Inode == 0
}

// isDigit returns true if the rune d represents an ascii digit
// between 0 and 9, inclusive.
func isDigit(d uint8) bool {
	return d >= '0' && d <= '9'
}

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

func splitSpaces(b []byte) [][]byte {
	res := make([][]byte, 0, 6)
	s := bytes.SplitN(b, []byte{' '}, 2)
	for len(s) > 1 {
		res = append(res, s[0])
		s = bytes.SplitN(bytes.TrimSpace(s[1]), []byte{' '}, 2)
	}
	res = append(res, s[0])
	return res
}

// procMem returns the amount of Pss, shared, and swapped out memory
// used.  The swapped out amount refers to anonymous pages only.
func procMem(pid int) (pss float64, shared, priv, swap int64, err error) {
	fPath := fmt.Sprintf("/proc/%d/smaps", pid)
	f, err := os.Open(fPath)
	if err != nil {
		err = fmt.Errorf("os.Open(%s): %s", fPath, err)
		return
	}
	//var curr MapInfo
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

		if bytes.Contains(l, []byte{'-'}) {
			//curr = NewMapInfo(l)
			continue
		}
		pieces := splitSpaces(l)
		ty := string(pieces[0])
		var v int64
		switch ty {
		case "Pss:":
			v, err = strconv.ParseInt(string(pieces[1]), 10, 64)
			if err != nil {
				err = fmt.Errorf("Atoi(%s): %s", string(pieces[1]), err)
				return
			}
			pss += float64(v) + PssAdjust
		case "Shared_Clean:", "Shared_Dirty:":
			v, err = strconv.ParseInt(string(pieces[1]), 10, 64)
			if err != nil {
				err = fmt.Errorf("Atoi(%s): %s", string(pieces[1]), err)
				return
			}
			shared += v
		case "Private_Clean:", "Private_Dirty:":
			v, err = strconv.ParseInt(string(pieces[1]), 10, 64)
			if err != nil {
				err = fmt.Errorf("Atoi(%s): %s", string(pieces[1]), err)
				return
			}
			priv += v
		case "Swap:":
			v, err = strconv.ParseInt(string(pieces[1]), 10, 64)
			if err != nil {
				err = fmt.Errorf("Atoi(%s): %s", string(pieces[1]), err)
				return
			}
			swap += v
		}
	}
	return
}

func worker(pidRequest chan int, wg *sync.WaitGroup, result chan *CmdMemInfo) {
	for pid := range pidRequest {
		var err error
		cmi := new(CmdMemInfo)

		cmi.PIDs = []int{pid}
		cmi.Name, err = procName(pid)
		if err != nil {
			log.Printf("procName(%d): %s", pid, err)
			wg.Done()
			continue
		} else if cmi.Name == "" {
			// XXX: This happens with kernel
			// threads. maybe warn? idk.
			wg.Done()
			continue
		}

		cmi.Pss, cmi.Shared, cmi.Private, cmi.Swapped, err = procMem(pid)
		if err != nil {
			log.Printf("procMem(%d): %s", pid, err)
			wg.Done()
			continue
		}

		result <- cmi
		wg.Done()
	}
}

type byTotal []*CmdMemInfo

func (c byTotal) Len() int           { return len(c) }
func (c byTotal) Less(i, j int) bool { return c[i].Pss < c[j].Pss }
func (c byTotal) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

func main() {
	nCPU := runtime.NumCPU()
	// give us as much parallelism as possible
	runtime.GOMAXPROCS(nCPU)

	if os.Geteuid() != 0 {
		fmt.Printf("FATAL: root required.\n")
		return
	}

	pids, err := pidList()
	if err != nil {
		log.Printf("pidList: %s", err)
		return
	}

	var wg sync.WaitGroup
	work := make(chan int, len(pids))
	result := make(chan *CmdMemInfo, len(pids))

	for i := 0; i < nCPU; i++ {
		go worker(work, &wg, result)
	}

	wg.Add(len(pids))
	for _, pid := range pids {
		work <- pid
	}
	wg.Wait()

	// aggregate similar processes by command name.
	cmdMap := map[string]*CmdMemInfo{}
loop:
	for {
		select {
		case c := <-result:
			n := c.Name
			if _, ok := cmdMap[n]; !ok {
				cmdMap[n] = c
				continue
			}
			cmdMap[n].PIDs = append(cmdMap[n].PIDs, c.PIDs...)
			cmdMap[n].Pss += c.Pss
			cmdMap[n].Private += c.Private
			cmdMap[n].Shared += c.Shared
			cmdMap[n].Swapped += c.Swapped
		default:
			break loop
		}
	}

	// extract map values to a slice so we can sort them
	cmds := make([]*CmdMemInfo, 0, len(cmdMap))
	for _, c := range cmdMap {
		cmds = append(cmds, c)
	}
	sort.Sort(byTotal(cmds))

	// keep track of total RAM and swap usage
	var totPss, totSwap float64

	fmt.Printf("%10s%10s%10s\t%s\n", "MB RAM", "PRIVATE", "SWAPPED", "PROCESS (COUNT)")
	for _, c := range cmds {
		n := c.Name
		if len(n) > CmdDisplayMax {
			n = n[:CmdDisplayMax]
		}
		s := ""
		if c.Swapped > 0 {
			swap := float64(c.Swapped)/1024.
			totSwap += swap
			s = fmt.Sprintf("%10.1f", swap)
		}
		pss := float64(c.Pss)/1024.
		fmt.Printf("%10.1f%10.1f%10s\t%s (%d)\n", pss, float64(c.Private)/1024, s, n, len(c.PIDs))
		totPss += pss
	}
	fmt.Printf("#%9.1f%20.1f\tTOTAL USED BY PROCESSES\n", totPss, totSwap)
}
