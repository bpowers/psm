package main

import (
	"os"
	"runtime"
	"runtime/pprof"
)

type ProfInstance struct {
	memprof, cpuprof *os.File
}

func NewProf(memprof, cpuprof string) (p *ProfInstance, err error) {
	p = &ProfInstance{}
	if memprof != "" {
		if p.memprof, err = os.Create(memprof); err != nil {
			p = nil
			return
		}
	}
	if cpuprof != "" {
		if p.cpuprof, err = os.Create(cpuprof); err != nil {
			// close all files on error
			if p.memprof != nil {
				p.memprof.Close()
			}
			p = nil
			return
		}
	}
	return
}

// startProfiling enables memory and/or CPU profiling if the
// appropriate command line flags have been set.
func (p *ProfInstance) Start() {

	// if we've passed in filenames to dump profiling data too,
	// start collecting profiling data.
	if p.memprof != nil {
		runtime.MemProfileRate = 1
	}
	if p.cpuprof != nil {
		pprof.StartCPUProfile(p.cpuprof)
	}
}

func (p *ProfInstance) Stop() {
	if p.memprof != nil {
		runtime.GC()
		pprof.WriteHeapProfile(p.memprof)
		p.memprof.Close()
	}
	if p.cpuprof != nil {
		pprof.StopCPUProfile()
		p.cpuprof.Close()
	}
}
