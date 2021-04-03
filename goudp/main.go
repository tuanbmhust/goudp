package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
	"unicode"

	"github.com/joho/godotenv"
)

func main() {
	//Read config from .env and setup
	err := godotenv.Load(".env")
	if err != nil {
		log.Panicf("Error loading .env file: %v", err)
		return
	}

	app := config{}
	app.connections, err = strconv.Atoi(os.Getenv("CONNECTIONS"))          //number of parallel connections
	app.host = os.Getenv("DEST_IP") + ":" + os.Getenv("DEST_PORT")         //Host IP
	app.defaultPort = ":8080"                                              //default port if missing listeners
	app.opt.UDPReadSize, err = strconv.Atoi(os.Getenv("READ_SIZE"))        //
	app.opt.UDPWriteSize, err = strconv.Atoi(os.Getenv("WRITE_SIZE"))      //
	app.opt.MaxSpeed, err = strconv.ParseFloat(os.Getenv("MAX_SPEED"), 64) //equals 0 means unlimited
	app.reportInterval = os.Getenv("REPORT_INTERVAL") + "s"                //time between 2 report
	app.totalDuration = os.Getenv("TOTAL_DURATION") + "s"                  //total report time
	app.numThreadSV, err = strconv.Atoi(os.Getenv("SV_MAX_PROCS"))         //Number of thread of server
	app.numThreadCL, err = strconv.Atoi(os.Getenv("CL_MAX_PROCS"))         //Number of thread of client
	app.localAddr = os.Getenv("SRC_IP") + ":" + os.Getenv("SRC_PORT")      //

	if err != nil {
		log.Panicf("Error while reading from .ENV file: %v", err)
		return
	}

	flag.BoolVar(&app.isClient, "client", false, "run in client mode")
	flag.BoolVar(&app.isOnlyReadServer, "readserver", true, "Set to `false` to use duplex server in server mode")
	flag.BoolVar(&app.isOnlyWriteClient, "readclient", true, "Set to `false` to use duplex client in client mode")

	flag.Parse()

	app.reportInterval = defaultTimeUnit(app.reportInterval)
	app.totalDuration = defaultTimeUnit(app.totalDuration)

	var errInterval error
	app.opt.ReportInterval, errInterval = time.ParseDuration(app.reportInterval)
	if errInterval != nil {
		log.Panicf("bad reportInterval: %q: %v", app.reportInterval, errInterval)
	}

	var errDuration error
	app.opt.TotalDuration, errDuration = time.ParseDuration(app.totalDuration)
	if errDuration != nil {
		log.Panicf("bad totalDuration: %q: %v", app.totalDuration, errDuration)
	}

	log.Printf("connections=%d defaultPort=%s hosts=%q", app.connections, app.defaultPort, app.host)

	// Run in server mode
	if app.isClient == false {
		log.Printf("In server mode... use '-client' to switch to client mode...")
		openServer(&app)
		return
	}

	//Run in client mode
	var proto string = "udp"
	log.Printf("In client mode, %s protocol", proto)
	openClient(&app, proto)
}

//Convert default time unit
func defaultTimeUnit(s string) string {
	if len(s) < 1 {
		return s
	}
	if unicode.IsDigit(rune(s[len(s)-1])) {
		return s + "s"
	}
	return s
}
