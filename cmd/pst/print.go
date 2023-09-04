package main

import (
	"fmt"
	"io"
)

func (r searchResult) print(w io.Writer) {
	for _, proc := range r.procs {
		fmt.Fprintf(w, "%+v\n", proc)
	}
}
