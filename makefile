DATE = $(shell date /t)

.PHONY: build
build:
	go build -o ../disk-usage-monitor_bin  disk-usage-monitor.go

.PHONY: git
git:
	git a 
	git co "${DATE}"
	git pusm
#	git pusn

.DEFAULT_GOAL := build