package main

import (
	"strconv"
)

// USD used to represent some amount in US cents.
type Cent int64

func(u Cent) Display() string {
	s := strconv.FormatInt(int64(u), 10)
	i := len(s) - 2
	return s[:i] + string('.') + s[i:]
}