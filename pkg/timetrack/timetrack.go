package timetrack

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

type TimeTracker struct {
	timerDeep int32
}

func (b *TimeTracker) Start(span string) func() {
	if b == nil {
		return func() {}
	}
	deep := atomic.AddInt32(&b.timerDeep, 1)
	n := time.Now()
	fmt.Printf("[timer]%s %s start\n", strings.Repeat(" ", int((deep-1)*2)), span)
	return func() {
		tc := time.Since(n)
		fmt.Printf("[timer]%s %s end %v\n", strings.Repeat(" ", int((deep-1)*2)), span, tc)
		atomic.AddInt32(&b.timerDeep, -1)
	}
}
