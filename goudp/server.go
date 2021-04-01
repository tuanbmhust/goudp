package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

func openServer(app *config) {
	var wg sync.WaitGroup

	host := appendPortIfMissing(app.host, app.defaultPort)
	listenUDP(app, &wg, host)

	wg.Wait()
}

func listenUDP(app *config, wg *sync.WaitGroup, h string) {
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

	wg.Add(1)
	handleUDP(app, wg, conn)
}

func handleUDP(app *config, wg *sync.WaitGroup, conn *net.UDPConn) {
	defer wg.Done()

	var idCount int //Count the number of src connect to server
	var aggReader aggregate
	var aggWriter aggregate

	tab := map[string]*udpInfo{}
	buf := make([]byte, app.opt.UDPReadSize)

	for {
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
			log.Printf("handleUDP: Receive from source: %v", src)

			info = &udpInfo{
				remote: src,
				acc:    &account{},
				start:  time.Now(),
				id:     idCount,
			}

			idCount++
			info.acc.prevTime = info.start
			tab[src.String()] = info

			dec := gob.NewDecoder(bytes.NewBuffer(buf[:n]))
			if errOpt := dec.Decode(&info.opt); errOpt != nil {
				log.Printf("handleUDP: ERROR options failure: %v", errOpt)
				//Remove error client
				delete(tab, src.String())
				info = nil
				continue
			}
			log.Printf("handleUDP: options received: %v", info.opt)

			if !app.isOnlyReadServer {
				go serverWriteTo(conn, info.opt, src, info.acc, info.id, 0, &aggWriter)
			}

			continue
		}
		//Receive msg from existed src
		connIndex := fmt.Sprintf("%d/%d", info.id, 0)

		if errRead != nil {
			log.Printf("handleUDP: ERROR while reading: connection index: %s from %s: %v", connIndex, src, errRead)
			continue
		}

		if time.Since(info.start) > info.opt.TotalDuration {
			log.Printf("handleUDP: total duration %s timer: %s", info.opt.TotalDuration, src)
			info.acc.average(info.start, connIndex, "handleUDP", "rcv/s", &aggReader)

			//Remove idle client from table
			delete(tab, src.String())
			info = nil

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

	workLoop(connIndex, "serverWriteTo", "snd/s", udpWriteTo, buf, opt.ReportInterval, opt.MaxSpeed, agg)

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
