package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
)

func isDigit(d uint8) bool {
	return d >= '0' && d <= '9'
}

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

func main() {
	// give us as much parallelism as possible
	runtime.GOMAXPROCS(runtime.NumCPU())

	if os.Geteuid() != 0 {
		fmt.Printf("FATAL: root required.")
		return
	}

	pids, err := pidList()
	if err != nil {
		log.Printf("pidList: %s", err)
		return
	}
	log.Printf("pids: %v", pids)
}
