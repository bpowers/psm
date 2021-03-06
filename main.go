package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
)

const (
	CmdDisplayMax = 32

	usage = `Usage: %s [OPTION...]
Simple, accurate RAM and swap reporting.

Options:
`
)

var (
	filter     string
	memProfile string
	cpuProfile string
	showHeap   bool
	filterRE   *regexp.Regexp
)

// store info about a command (group of processes), similar to how
// ps_mem works.
type CmdMemInfo struct {
	PIDs    []int
	Name    string
	Pss     float64
	Shared  float64
	Heap    float64
	Swapped float64
}

type MapInfo struct {
	Inode uint64
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
	if len(pieces) < 5 {
		panic(fmt.Sprintf("NewMapInfo(%d): `%s`",
			len(pieces), string(mapLine)))
	}
	mi.Inode, err = ParseUint(pieces[4], 10, 64)
	if err != nil {
		panic(fmt.Sprintf("NewMapInfo: Atoi(%s): %s (%s)",
			string(pieces[4]), err, string(mapLine)))
	}
	return mi
}

func (mi MapInfo) IsAnon() bool {
	return mi.Inode == 0
}

// worker is executed in a new goroutine.  Its sole purpose is to
// process requests for information about particular PIDs.
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
		} else if filterRE != nil && !filterRE.MatchString(cmi.Name) {
			wg.Done()
			continue
		}

		cmi.Pss, cmi.Shared, cmi.Heap, cmi.Swapped, err = procMem(pid)
		if err != nil {
			log.Printf("procMem(%d): %s", pid, err)
			wg.Done()
			continue
		}

		result <- cmi
		wg.Done()
	}
}

type byPss []*CmdMemInfo

func (c byPss) Len() int           { return len(c) }
func (c byPss) Less(i, j int) bool { return c[i].Pss < c[j].Pss }
func (c byPss) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&filter, "filter", "",
		"regex to test process names against")
	flag.StringVar(&memProfile, "memprofile", "",
		"write memory profile to this file")
	flag.StringVar(&cpuProfile, "cpuprofile", "",
		"write cpu profile to this file")
	flag.BoolVar(&showHeap, "heap", false, "show heap column")

	flag.Parse()

	if filter != "" {
		filterRE = regexp.MustCompile(filter)
	}
}

func main() {
	prof, err := NewProf(memProfile, cpuProfile)
	if err != nil {
		log.Fatal(err)
	}
	// if -memprof or -cpuprof haven't been set on the command
	// line, these are nops
	prof.Start()
	defer prof.Stop()

	// need to be root to read map info for other user's
	// processes.
	if os.Geteuid() != 0 {
		fmt.Printf("%s requires root privileges. (try 'sudo `which %s`)\n",
			os.Args[0], os.Args[0])
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

	// give us as much parallelism as possible
	nCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nCPU)
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
		// this only works correctly because we a channel
		// where the buffer size >= the number of potential
		// results.
		select {
		case c := <-result:
			n := c.Name
			if _, ok := cmdMap[n]; !ok {
				cmdMap[n] = c
				continue
			}
			cmdMap[n].PIDs = append(cmdMap[n].PIDs, c.PIDs...)
			cmdMap[n].Pss += c.Pss
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
	sort.Sort(byPss(cmds))

	// keep track of total RAM and swap usage
	var totPss, totSwap float64

	headFmt := "%10s%10s%10s\t%s\n"
	cols := []interface{}{"MB RAM", "SHARED", "SWAPPED", "PROCESS (COUNT)"}
	totFmt := "#%9.1f%20.1f\tTOTAL USED BY PROCESSES\n"

	if showHeap {
		headFmt = "%10s" + headFmt
		cols = []interface{}{"MB RAM", "SHARED", "HEAP", "SWAPPED", "PROCESS (COUNT)"}
		totFmt = "#%9.1f%30.1f\tTOTAL USED BY PROCESSES\n"
	}

	fmt.Printf(headFmt, cols...)
	for _, c := range cmds {
		n := c.Name
		if len(n) > CmdDisplayMax {
			if n[0] == '[' {
				n = n[:strings.IndexRune(n, ']')+1]
			} else {
				n = n[:CmdDisplayMax]
			}
		}
		s := ""
		if c.Swapped > 0 {
			swap := c.Swapped / 1024.
			totSwap += swap
			s = fmt.Sprintf("%10.1f", swap)
		}
		pss := float64(c.Pss) / 1024.
		if showHeap {
			fmt.Printf("%10.1f%10.1f%10.1f%10s\t%s (%d)\n", pss, c.Shared/1024., c.Heap/1024., s, n, len(c.PIDs))
		} else {
			fmt.Printf("%10.1f%10.1f%10s\t%s (%d)\n", pss, c.Shared/1024., s, n, len(c.PIDs))
		}
		totPss += pss
	}
	fmt.Printf(totFmt, totPss, totSwap)
}
