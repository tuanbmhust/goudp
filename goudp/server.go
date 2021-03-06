package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"runtime"
	"time"
)

func openServer(app *config) {
	//Number of CPU can use
	runtime.GOMAXPROCS(app.numProcSV)

	log.Printf("%v", app)

	host := appendPortIfMissing(app.host, app.defaultPort)
	listenUDP(app, host)
}

func listenUDP(app *config, h string) {
	log.Printf("Server: spawning UDP listener: %s", h)

	udpAddr, errAddr := net.ResolveUDPAddr("udp", h)
	if errAddr != nil {
		log.Printf("listenUDP: ERROR bad udp address: %s: %v", h, errAddr)
		return
	}

	conn, errConn := net.ListenUDP("udp", udpAddr)
	if errConn != nil {
		log.Printf("listenUDP: ERROR listen: %s: %v", h, errConn)
		return
	}

	handleUDP(app, conn)
}

func handleUDP(app *config, conn *net.UDPConn) {

	var idCount int //Count the number of src connect to server
	var aggReader aggregate
	var aggWriter aggregate

	tab := map[string]*udpInfo{}
	buf := make([]byte, app.opt.UDPReadSize)

	for {
		for key, val := range tab { //Check if exist an overtime active client then remove
			if time.Since(val.start) >= val.opt.TotalDuration {
				connIndex := fmt.Sprintf("%d/%d", val.id, 0)
				val.acc.average(val.start, connIndex, "handleUDP", "rcv/s", &aggReader)
				log.Printf("Total packet Server received from %s: %d", key, val.acc.calls)
				delete(tab, key)
				idCount--
				continue
			}
		}

		var info *udpInfo
		n, src, errRead := conn.ReadFromUDP(buf) //Read from client

		if src == nil {
			log.Printf("handleUDP: ERROR read nil src: %v", errRead)
			continue
		}

		var found bool
		info, found = tab[src.String()]

		//If the connection is from new src
		if !found {
			info = &udpInfo{
				remote: src,
				acc:    &account{},
				start:  time.Now(),
				id:     idCount,
			}

			//Remove packet that is not from new src after conn.Close() which has null opt
			dec := gob.NewDecoder(bytes.NewBuffer(buf[:n]))
			if errOpt := dec.Decode(&info.opt); errOpt != nil {
				//log.Printf("handleUDP: ERROR options: %v", errOpt)
				continue
			}

			idCount++
			info.acc.prevTime = info.start
			tab[src.String()] = info

			log.Printf("handleUDP: Receive from: %v", src)

			if !app.isOnlyReadServer {
				opt := info.opt
				go serverWriteTo(conn, opt, src, info.acc, info.id, 0, &aggWriter)
			}

			continue
		}

		//Receive msg from existed src
		connIndex := fmt.Sprintf("%d/%d", info.id, 0)

		if errRead != nil {
			log.Printf("handleUDP: ERROR while reading: connection index: %s from %s: %v", connIndex, src, errRead)
			continue
		}

		if time.Since(info.start) >= info.opt.TotalDuration {
			info.acc.average(info.start, connIndex, "handleUDP", "rcv/s", &aggReader)
			log.Printf("Total packet Server received from %s: %d", src, info.acc.calls)

			//Remove idle client from table
			delete(tab, src.String())
			info = nil
			idCount--
			continue
		}

		info.acc.update(n, info.opt.ReportInterval, connIndex, "handleUDP", "rcv/s")
	}
}

func serverWriteTo(conn *net.UDPConn, opt options, dest net.Addr, acc *account, id, connections int, agg *aggregate) {
	log.Printf("serverWriteTo: UDP %v", dest)

	start := acc.prevTime

	udpWriteTo := func(b []byte) (int, error) {
		if time.Since(start) > opt.TotalDuration {
			return -1, fmt.Errorf("udpWriteTo: total duration %s timer", opt.TotalDuration)
		}
		return conn.WriteTo(b, dest)
	}

	connIndex := fmt.Sprintf("%d/%d", id, connections)

	buf := randBuf(opt.UDPWriteSize)

	workLoop(connIndex, "Server write", "snd/s", udpWriteTo, buf, opt.ReportInterval, opt.MaxSpeed, agg)

	log.Printf("serverWriteTo: exiting: %v", dest)
}

func appendPortIfMissing(host, port string) string {
LOOP:
	for i := len(host) - 1; i >= 0; i-- {
		c := host[i]
		switch c {
		case ']':
			break LOOP
		case ':':
			return host
		}
	}

	return host + port
}
