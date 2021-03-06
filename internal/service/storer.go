package service

import (
	"context"

	"redits.oculeus.com/asorokin/disk-space-monitor-src/internal/datastructs"
)

type storer interface {
	CheckRelation(ctx context.Context) error
	SavedDisk(ctx context.Context, serverName string) ([]datastructs.DiskInfo, error)
	UpdateInfo(ctx context.Context, serverName string, info map[string]datastructs.DiskInfo) error
}
