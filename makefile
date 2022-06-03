VERSION = 2.0.1
DATE = $(shell date /t)

.PHONY: build
build:
	go build -ldflags "-X main.version=${VERSION}" -o ../disk-space-monitor  disk-space-monitor.go

.PHONY: git
git:
	git a 
	git co "${DATE}"
	git pusm
#	git pusn

.DEFAULT_GOAL := build