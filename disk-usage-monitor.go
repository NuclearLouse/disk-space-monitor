package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	threshold = flag.Int("threshold", 90, "used disk space threshold in %")
	checktime = flag.Int64("checktime", 60, "check period in minutes")
)

func main() {
	flag.Parse()

	logFile, err := os.OpenFile("disk-usage-monitor.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	logger := log.New(logFile, "[WARNING] ", log.Ldate|log.Ltime|log.Lmsgprefix)

	for {
		cmd := exec.Command("sh", "-c", "df / | grep / | awk '{ print $5}' | sed 's/%//g'")
		bts, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}

		used, err := strconv.Atoi(strings.TrimSuffix(string(bts), "\n"))
		if err != nil {
			log.Fatal(err)
		}

		if used > *threshold {
			logger.Printf("used disk space: %d%% - threshold exceeded!\n", used)
		}

		time.Sleep(time.Duration(*checktime) * time.Minute)
	}

}
