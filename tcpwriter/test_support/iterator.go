package test_support

import "time"

// TimeIterator returns a function returning the defined values in order
// if no more values are defined the last one is used
func TimeIterator(times ...time.Time) func() time.Time {
	if len(times) == 0 {
		panic("no values are defined, need at least one")
	}
	index := 0
	return func() time.Time {
		if index > len(times)-1 {
			return times[index-1]
		}
		now := times[index]
		index += 1
		return now
	}
}
