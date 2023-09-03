package main

import (
	"regexp"
)

func parseArgs(args []string) (c config, _ error) {

	for _, a := range args {
		rx, err := regexp.Compile(a)
		if err != nil {
			return c, err
		}
		c.filters = append(c.filters, rx.MatchString)
	}

	return c, nil
}
