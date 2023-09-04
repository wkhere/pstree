// Copyright 2015 The pstree Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pstree provides an API to retrieve the process tree from procfs.
package pstree // import "github.com/sbinet/pstree"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Tree struct {
	Procs map[int]Process `json:"procs"`
}

type Process struct {
	Name     string      `json:"name"`
	Stat     ProcessStat `json:"stat"`
	Children []int       `json:"children"`
}

// New returns the whole system process tree.
func New() (*Tree, error) {
	files, err := filepath.Glob("/proc/[0-9]*")
	if err != nil {
		return nil, fmt.Errorf("could not list pid files under /proc: %w", err)
	}

	procs := make(map[int]Process, len(files))
	for _, dir := range files {
		proc, err := scan(dir)
		if err != nil {
			return nil, fmt.Errorf("could not scan %s: %w", dir, err)
		}
		if proc.Stat.PID == 0 {
			// process vanished since Glob.
			continue
		}
		procs[proc.Stat.PID] = proc
	}

	for pid, proc := range procs {
		if proc.Stat.Ppid == 0 {
			continue
		}
		parent, ok := procs[proc.Stat.Ppid]
		if !ok {
			return nil, fmt.Errorf("parent pid=%d of pid=%d does not exist",
				proc.Stat.Ppid, pid,
			)
		}
		parent.Children = append(parent.Children, pid)
		procs[parent.Stat.PID] = parent
	}

	for pid, proc := range procs {
		if len(proc.Children) > 0 {
			sort.Ints(proc.Children)
		}
		procs[pid] = proc
	}

	return &Tree{Procs: procs}, err
}

// ProcessStat contains process information.
// see: http://man7.org/linux/man-pages/man5/proc.5.html
type ProcessStat struct {
	PID       int    `json:"pid"`       // process ID
	Comm      string `json:"comm"`      // command, like /proc/$Pid/comm but found in stat
	State     byte   `json:"state"`     // process state
	Ppid      int    `json:"ppid"`      // pid of the parent process
	Pgrp      int    `json:"pgrp"`      // process group ID of the process
	Session   int    `json:"session"`   // session ID of the process
	TTY       int    `json:"tty"`       // controlling terminal of the process
	Tpgid     int    `json:"tpgid"`     // ID of foreground process group
	Flags     uint32 `json:"flags"`     // kernel flags word of the process
	Minflt    uint64 `json:"minflt"`    // number of minor faults the process has made which have not required loading a memory page from disk
	Cminflt   uint64 `json:"cminflt"`   // number of minor faults the process's waited-for children have made
	Majflt    uint64 `json:"majflt"`    // number of major faults the process has made which have required loading a memory page from disk
	Cmajflt   uint64 `json:"cmajflt"`   // number of major faults the process's waited-for children have made
	Utime     uint64 `json:"utime"`     // user time in clock ticks
	Stime     uint64 `json:"stime"`     // system time in clock ticks
	Cutime    int64  `json:"cutime"`    // children user time in clock ticks
	Cstime    int64  `json:"cstime"`    // children system time in clock ticks
	Priority  int64  `json:"priority"`  // priority
	Nice      int64  `json:"nice"`      // the nice value
	Nthreads  int64  `json:"nthreads"`  // number of threads in this process
	Itrealval int64  `json:"itrealval"` // time in jiffies before next SIGALRM is sent to the process due to an interval timer
	Starttime int64  `json:"starttime"` // time the process started after system boot in clock ticks
	Vsize     uint64 `json:"vsize"`     // virtual memory size in bytes
	RSS       int64  `json:"rss"`       // resident set size: number of pages the process has in real memory

	Environ string `json:"environ"` // environment; k=v pairs separated by byte 0
	Cwd     string `json:"cwd"`     // current working directory for the process
	Cmdline string `json:"cmdline"` // complete command line; args separated by byte 0
}

func scan(dir string) (Process, error) {
	const (
		// statfmt is the stat format as described in proc.5.html
		// note that the first 2 fields "pid" and "(comm)" are dealt with separately
		// and are thus not specified in statfmt below.
		statfmt = "%c %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d"
	)

	stat := filepath.Join(dir, "stat")
	data, err := ioutil.ReadFile(stat)
	if err != nil {
		// process vanished since Glob.
		return Process{}, nil
	}
	// extracting the name of the process, enclosed in matching parentheses.
	info := strings.FieldsFunc(string(data), func(r rune) bool {
		return r == '(' || r == ')'
	})

	if len(info) != 3 {
		return Process{}, fmt.Errorf("%s: file format invalid", stat)
	}

	for i, v := range info {
		info[i] = strings.TrimSpace(v)
	}

	var proc Process
	proc.Stat.PID, err = strconv.Atoi(info[0])
	if err != nil {
		return Process{}, fmt.Errorf("%s: invalid pid format %q: %w", stat, info[0], err)
	}
	proc.Stat.Comm = info[1]

	_, err = fmt.Sscanf(
		info[2], statfmt,
		&proc.Stat.State,
		&proc.Stat.Ppid, &proc.Stat.Pgrp, &proc.Stat.Session,
		&proc.Stat.TTY, &proc.Stat.Tpgid, &proc.Stat.Flags,
		&proc.Stat.Minflt, &proc.Stat.Cminflt, &proc.Stat.Majflt, &proc.Stat.Cmajflt,
		&proc.Stat.Utime, &proc.Stat.Stime,
		&proc.Stat.Cutime, &proc.Stat.Cstime,
		&proc.Stat.Priority,
		&proc.Stat.Nice,
		&proc.Stat.Nthreads,
		&proc.Stat.Itrealval, &proc.Stat.Starttime,
		&proc.Stat.Vsize, &proc.Stat.RSS,
	)
	if err != nil {
		return proc, fmt.Errorf("could not parse file %s: %w", stat, err)
	}

	env, err := os.ReadFile(filepath.Join(dir, "environ"))
	switch {
	case errors.Is(err, os.ErrPermission):
		// ignore
	case err != nil:
		return proc, fmt.Errorf("could not read environ: %w", err)
	}
	proc.Stat.Environ = string(env)

	cwd := filepath.Join(dir, "cwd")
	pwd, err := os.Readlink(cwd)
	switch {
	case errors.Is(err, os.ErrPermission):
		// ignore
	case err != nil:
		return proc, fmt.Errorf("could not stat %s: %w", cwd, err)
	}
	proc.Stat.Cwd = pwd

	cmd, err := os.ReadFile(filepath.Join(dir, "cmdline"))
	if err != nil {
		return proc, fmt.Errorf("could not read cmdline: %w", err)
	}
	proc.Stat.Cmdline = string(cmd)

	proc.Name = proc.Stat.Comm
	return proc, nil
}
