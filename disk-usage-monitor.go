package main

import (
	"flag"
	"fmt"
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
	info      = flag.Bool("v", false, "will display the version of the program")
	version   string
)

func main() {
	flag.Parse()
	if *info {
		showVersion()
		return
	}
	logFile, err := os.OpenFile("disk-usage-monitor.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	warn := log.New(logFile, "[WARNING] ", log.Ldate|log.Ltime|log.Lmsgprefix)
	info := log.New(logFile, "[INFO] ", log.Ldate|log.Ltime|log.Lmsgprefix)

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
			warn.Printf("used disk space: %d%% - threshold exceeded!\n", used)
		} else {
			info.Printf("used disk space: %d%%\n", used)
		}

		time.Sleep(time.Duration(*checktime) * time.Minute)
	}

}

func showVersion() {
	fmt.Println("Version=", version)
}
