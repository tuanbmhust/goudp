package main

import (
	"net"
	"sync"
	"time"
)

type hostList []string

type config struct {
	hosts             hostList
	listeners         hostList
	defaultPort       string
	connections       int
	reportInterval    string
	totalDuration     string
	opt               options
	chart             string
	localAddr         string
	isClient          bool
	isOnlyReadServer  bool
	isOnlyWriteClient bool
}

type options struct {
	ReportInterval time.Duration
	TotalDuration  time.Duration
	UDPReadSize    int     //byte
	UDPWriteSize   int     //byte
	MaxSpeed       float64 //mbps
}

type aggregate struct {
	Mbps  int64 // Megabit/s
	Cps   int64 // Call/s
	mutex sync.Mutex
}

type udpInfo struct {
	remote *net.UDPAddr
	opt    options
	acc    *account
	start  time.Time
	id     int
}

type call func(p []byte) (n int, err error)
