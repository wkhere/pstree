// Copyright 2015 The pstree Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command procs-tree displays the tree of children processes for a given PID.
package main

import (
	"fmt"
	"os"

	pst "github.com/wkhere/pstree"
)

type config struct {
	filters []filter
}

var selfPID = os.Getpid()

func main() {
	conf, err := parseArgs(os.Args[1:])
	if err != nil {
		die(2, err)
	}

	tree, err := pst.New(pst.Options{})
	if err != nil {
		die(1, "could not read process tree:", err)
	}

	proc := bfs(conf.filters, tree, tree.Procs[1])
	fmt.Printf("%+v\n", proc) //tmp
}

func die(exitcode int, a ...any) {
	if len(a) > 0 {
		fmt.Fprintln(os.Stderr, a...)
	}
	os.Exit(exitcode)
}
