package main

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

func openClient(app *config, proto string) {
	//Number of CPU can use
	runtime.GOMAXPROCS(app.numProcCL)

	var wg sync.WaitGroup
	var aggReader aggregate
	var aggWriter aggregate

	dialer := net.Dialer{}

	if app.localAddr != "" {
		addr, err := net.ResolveUDPAddr(proto, app.localAddr)
		if err != nil {
			log.Printf("openClient: error while resolving udp address=%s: %v", app.localAddr, err)
		}
		dialer.LocalAddr = addr
	}

	host := appendPortIfMissing(app.host, app.defaultPort)
	for i := 0; i < app.connections; i++ {
		conn, err := dialer.Dial(proto, host)

		if err != nil {
			log.Printf("openClient: error while dial host %s: %v", host, err)
			continue
		}

		spawnClient(app, &wg, conn, i, app.connections, &aggReader, &aggWriter)
	}

	wg.Wait()

	log.Printf("aggregate writing: %d Mbps %d send/s", aggWriter.Mbps, aggWriter.Cps)
	if !app.isOnlyWriteClient {
		log.Printf("aggregate reading: %d Mbps %d recv/s", aggReader.Mbps, aggReader.Cps)
	}
}

func spawnClient(app *config, wg *sync.WaitGroup, conn net.Conn, cl, connection int, aggReader, aggWriter *aggregate) {
	wg.Add(1)
	go handleConnectionClient(app, wg, conn, cl, connection, aggReader, aggWriter)
}

func sendOptions(app *config, conn net.Conn) error {
	var optBuf bytes.Buffer
	enc := gob.NewEncoder(&optBuf)
	opt := app.opt

	if err := enc.Encode(&opt); err != nil {
		log.Printf("handleConnectionClient: UDP options failure: %v", err)
		return err
	}

	if _, err := conn.Write(optBuf.Bytes()); err != nil {
		log.Printf("handleConnectionClient: UDP options write: %v", err)
		return err
	}

	return nil
}

func handleConnectionClient(app *config, wg *sync.WaitGroup, conn net.Conn, cl, connection int, aggReader, aggWriter *aggregate) {
	defer wg.Done()

	// log.Printf("handleConnectionClient: starting %d/%d: from %v to %v", cl, connection, conn.LocalAddr(), conn.RemoteAddr())
	if err := sendOptions(app, conn); err != nil {
		return
	}
	log.Printf("handleConnectionClient: option sent: %v", app.opt)

	doneReader := make(chan struct{})
	doneWriter := make(chan struct{})

	opt := app.opt

	bufSizeOut := app.opt.UDPWriteSize
	go clientWriter(conn, cl, connection, doneWriter, bufSizeOut, opt, aggWriter)

	if !app.isOnlyWriteClient {
		bufSizeIn := app.opt.UDPReadSize
		go clientReader(conn, cl, connection, doneReader, bufSizeIn, opt, aggReader)
	}

	timerPeriod := time.NewTimer(app.opt.TotalDuration)
	<-timerPeriod.C
	timerPeriod.Stop()

	conn.Close()

	<-doneWriter
	if !app.isOnlyWriteClient {
		<-doneReader
	}

	// log.Printf("handleConnectionClient: stopping udp %d/%d: %v", cl, connection, conn.RemoteAddr())
}

func clientReader(conn net.Conn, cl, connection int, doneReader chan struct{}, bufSize int, opt options, agg *aggregate) {
	// log.Printf("clientReader: starting id %d/%d %v", cl, connection, conn.RemoteAddr())

	connIndex := fmt.Sprintf("%d/%d", cl, connection)
	buf := make([]byte, bufSize)
	workLoop(connIndex, "Client received", "rcv/s", conn.Read, buf, opt.ReportInterval, 0, agg)
	close(doneReader)

	// log.Printf("clientReader: stopping id %d/%d %v", cl, connection, conn.RemoteAddr())
}

func clientWriter(conn net.Conn, cl, connection int, doneWriter chan struct{}, bufSize int, opt options, agg *aggregate) {
	// log.Printf("clientWriter: starting id %d/%d %v", cl, connection, conn.RemoteAddr())

	connIndex := fmt.Sprintf("%d/%d", cl, connection)
	buf := randBuf(bufSize)
	workLoop(connIndex, "Client sent", "snd/s", conn.Write, buf, opt.ReportInterval, opt.MaxSpeed, agg)
	close(doneWriter)

	// log.Printf("clientWriter: stopping id %d/%d %v", cl, connection, conn.RemoteAddr())
}

//workLoop do the loop for each go routine
func workLoop(connID, label, cpsLabel string, f call, buf []byte, reportInterval time.Duration, maxSpeed float64, agg *aggregate) {
	start := time.Now()
	acc := &account{}
	acc.prevTime = start

	for {
		runtime.Gosched()

		if maxSpeed > 0 {
			elapSec := time.Since(acc.prevTime).Seconds()
			if elapSec > 0 {
				mbps := float64(8*(acc.size-acc.prevSize)) / (1000000 * elapSec) //Megabits per second
				if mbps > maxSpeed {
					time.Sleep(time.Microsecond)
					continue
				}
			}
		}

		n, err := f(buf)
		if err != nil {
			//log.Printf("workLoop %s %s: %v", connID, label, err)
			break
		}

		acc.update(n, reportInterval, connID, label, cpsLabel)

	}

	acc.average(start, connID, label, cpsLabel, agg)
	log.Printf("Total packet "+label+": %d", acc.calls)
}

func randBuf(size int) []byte {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		log.Printf("randBuf: ERROR %v", err)
	}

	return buf
}
