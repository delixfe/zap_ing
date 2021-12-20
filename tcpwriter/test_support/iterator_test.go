package test_support

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTimeIterator(t *testing.T) {
	now := time.Now()
	day := time.Hour * 24
	times := []time.Time{
		now,
		now.Add(day * 1),
		now.Add(day * 2),
		now.Add(day * 3),
	}
	tests := []struct {
		name string
		args []time.Time
		want []time.Time
	}{
		{name: "oneValue_returnsAlwaysTheSame", args: times[:1], want: []time.Time{now, now, now}},
		{name: "allValues_returnsTheseValues", args: times, want: times},
		{name: "twoValues_thirdReturnsSecond", args: []time.Time{times[0], times[1]}, want: []time.Time{times[0], times[1]}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nowFn := TimeIterator(tt.args...)
			for i, expected := range tt.want {
				actual := nowFn()
				assert.Equal(t, expected, actual, "%d invocation", i)
			}
		})
	}
}
