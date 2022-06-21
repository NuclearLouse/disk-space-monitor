VERSION=3.0.1

LDFLAGS = -ldflags "-X redits.oculeus.com/asorokin/disk-space-monitor-src-new/internal/service.version=${VERSION} -X redits.oculeus.com/asorokin/disk-space-monitor-src-new/internal/service.configFile=.yaml"
LDFLAGS_DEV = -ldflags "-X redits.oculeus.com/asorokin/disk-space-monitor-src-new/internal/service.version=${VERSION} -X redits.oculeus.com/asorokin/disk-space-monitor-src-new/internal/service.configFile=dev.yaml"
DATE = $(shell date /t)

.PHONY: build
build:
	go build ${LDFLAGS_DEV} -v ./cmd/disk-space-monitor

.PHONY: deploy
deploy:
	go build ${LDFLAGS} -v -o ../disk-space-monitor-new ./cmd/disk-space-monitor

.PHONY: git
git:
	git a 
	git co "${DATE}"
	git pusm
#	git pusn

.DEFAULT_GOAL := build