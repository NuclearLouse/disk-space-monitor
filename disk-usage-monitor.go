package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/bndr/gotabulate"
)

var (
	threshold = flag.Int("threshold", 90, "used disk space threshold in %")
	checktime = flag.Int64("checktime", 60, "check period in minutes")
	vers      = flag.Bool("v", false, "will display the version of the program")
	version   string

	warn, info *log.Logger
)

type diskInfo struct {
	Filesystem string
	Size       string
	Used       string
	Avail      string
	UsePrc     string
	MountedOn  string
}

func init() {
	logFile, err := os.OpenFile("disk-usage-monitor.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	warn = log.New(logFile, "[WARNING] ", log.Ldate|log.Ltime|log.Lmsgprefix)
	info = log.New(logFile, "[ INFO  ] ", log.Ldate|log.Ltime|log.Lmsgprefix)
}

func main() {
	flag.Parse()

	if *vers {
		showVersion()
		return
	}

	for {
		cmd := exec.Command("sh", "-c", "df -h")
		output, _ := cmd.StdoutPipe()
		defer output.Close()

		cmd.Start()

		scanner := bufio.NewScanner(output)
		scanner.Split(bufio.ScanLines)

		var (
			i     int
			infos []*diskInfo
		)

		for scanner.Scan() {
			line := scanner.Text()
			if i != 0 {
				row := strings.Split(strings.TrimSuffix(line, "\n"), " ")
				var newRow []string
				for _, w := range row {
					e := strings.TrimSpace(w)
					if len(e) == 0 {
						continue
					}
					newRow = append(newRow, e)
				}

				infos = append(infos, &diskInfo{newRow[0], newRow[1], newRow[2], newRow[3], newRow[4], newRow[5]})
			}
			i++
		}

		output.Close()

		cmd.Wait()

		var rows [][]string

		var used bool
		for _, i := range infos {
			useprc, err := strconv.Atoi(strings.TrimSuffix(i.UsePrc, "%"))
			if err != nil {
				log.Fatal(err)
			}
			if useprc >= *threshold {
				used = true
				rows = append(rows, []string{
					i.Filesystem,
					i.Size,
					i.Used,
					i.Avail,
					i.UsePrc,
					cut(i.MountedOn, 30),
				})
			}
		}
		if used {
			tab := gotabulate.Create(rows)
			tab.SetHeaders([]string{"Filesystem", "Size", "Used", "Available", "Use %", "Mounted on"})
			tab.SetWrapStrings(true)
			tab.SetAlign("center")
			warn.Printf("threshold exceeded!\n%s", tab.Render("grid"))
			fmt.Printf("threshold exceeded!\n%s", tab.Render("grid"))
		} else {
			info.Println("disk space free")
		}

		time.Sleep(time.Duration(*checktime) * time.Minute)
	}

}

func cut(text string, limit int) string {
	runes := []rune(text)
	if len(runes) >= limit {
		return string(runes[:limit])
	}
	return text
}

func showVersion() {
	fmt.Println("Version=", version)
}
