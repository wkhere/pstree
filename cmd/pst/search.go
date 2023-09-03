package main

import pst "github.com/wkhere/pstree"

type filter func(string) bool

func match(filters []filter, proc *pst.Process) bool {
	for _, f := range filters {
		if !f(proc.Stat.Cmdline) {
			return false
		}
	}
	return proc.Stat.PID != selfPID
}

func bfs(filters []filter, tree *pst.Tree, proc *pst.Process) *pst.Process {
	q := make(queue, 0, len(tree.Procs))

	pid := proc.Stat.PID
	q.push(pid)

	for len(q) > 0 {
		pid = q.pop()
		proc = tree.Procs[pid]

		if match(filters, proc) {
			return proc
		}
		for _, child := range proc.Children {
			q.push(child)
		}
	}
	return nil
}

type queue []int

func (q *queue) push(x int) { *q = append(*q, x) }
func (q *queue) pop() (x int) {
	x = (*q)[0]
	*q = (*q)[1:]
	return x
}
