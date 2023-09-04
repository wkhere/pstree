package main

import pst "github.com/wkhere/pstree"

type filter func(string) bool

type searchResult struct {
	tree  *pst.Tree
	procs []*pst.Process
}

func match(filters []filter, proc *pst.Process) bool {
	for _, f := range filters {
		if !f(proc.Stat.Cmdline) {
			return false
		}
	}
	return proc.Stat.PID != selfPID
}

func bfs(filters []filter, tree *pst.Tree, proc *pst.Process) searchResult {
	q := make(queue, 0, len(tree.Procs))
	r := searchResult{tree, make([]*pst.Process, 0, 4)} // cap=4 is a guess

	pid := proc.Stat.PID
	q.push(pid)

	for len(q) > 0 {
		pid = q.pop()
		proc = tree.Procs[pid]

		if match(filters, proc) {
			r.procs = append(r.procs, proc)
			continue
			// not descending here - finding only topmost processes matching
		}
		for _, child := range proc.Children {
			q.push(child)
		}
	}
	return r
}

type queue []int

func (q *queue) push(x int) { *q = append(*q, x) }
func (q *queue) pop() (x int) {
	x = (*q)[0]
	*q = (*q)[1:]
	return x
}
