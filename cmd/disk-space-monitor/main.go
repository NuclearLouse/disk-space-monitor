package main

import (
	"flag"
	"log"

	"redits.oculeus.com/asorokin/disk-space-monitor-src/internal/service"
)

func main() {
	info := flag.Bool("v", false, "will display the version of the program")
	direct := flag.Bool("d", false, "direct start of the service, then the settings will be taken from previously saved")
	flag.Parse()
	if *info {
		service.Version()
		return
	}
	service, err := service.New()
	if err != nil {
		log.Fatal("init new fluent-bit-control service:", err)
	}

	service.Start(*direct)
}