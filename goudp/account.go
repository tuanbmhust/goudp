package main

import (
	"log"
	"sync"
	"time"
)

type account struct {
	prevTime  time.Time
	prevSize  int64
	prevCalls int
	size      int64
	calls     int
	mutex     sync.Mutex
}

const reportFormat = "%s %7s %14s rate: %6d Mbps %6d %s"

func (a *account) average(start time.Time, conn, label, cpsLabel string, agg *aggregate) {
	elapSec := time.Since(start).Seconds()
	mbps := int64(float64(8*a.size) / (1000000 * elapSec)) //Megabits per second
	cps := int64(float64(a.calls) / elapSec)

	log.Printf(reportFormat, conn, "average", label, mbps, cps, cpsLabel)

	agg.mutex.Lock()
	agg.Mbps += mbps
	agg.Cps += cps
	agg.mutex.Unlock()
}

func (a *account) update(n int, reportInterval time.Duration, conn, label, cpsLabel string) {
	// a.mutex.Lock()
	a.calls++
	a.size += int64(n)
	// a.mutex.Unlock()

	now := time.Now()
	elap := now.Sub(a.prevTime)

	if elap > reportInterval {
		elapSec := elap.Seconds()
		mbps := int64(float64(8*(a.size-a.prevSize)) / (1000000 * elapSec)) //Megabits per second
		cps := int64(float64(a.calls-a.prevCalls) / elapSec)

		log.Printf(reportFormat, conn, "report", label, mbps, cps, cpsLabel)

		a.mutex.Lock()
		a.prevTime = now
		a.prevSize = a.size
		a.prevCalls = a.calls
		a.mutex.Unlock()
	}
}
