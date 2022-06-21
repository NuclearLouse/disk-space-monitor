package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"redits.oculeus.com/asorokin/disk-space-monitor-src/internal/datastructs"
)

type Postgres struct {
	*pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool}
}

// Server | Filesystem         |   Size  |   Used  |   Available  |   Use %  |   Mounted on | Treshold

func (db *Postgres) UpdateInfo(ctx context.Context, serverName string, info []datastructs.DiskInfo) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		"DELETE FROM disk_monitor.monitoring_disk WHERE server=$1", serverName); err != nil {
		return err
	}
	for _, i := range info {
		if _, err := tx.Exec(ctx,
			`INSERT INTO disk_monitor.monitoring_disk VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			serverName,
			i.Filesystem,
			i.Size,
			i.Used,
			i.Avail,
			i.UsePrc,
			i.MountedOn,
			i.Threshold,
			i.LastCheck,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (db *Postgres) SavedDisk(ctx context.Context, serverName string) ([]datastructs.DiskInfo, error) {
	rows, err := db.Query(ctx,
		"SELECT * FROM disk_monitor.monitoring_disk WHERE server=$1", serverName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disks []datastructs.DiskInfo
	for rows.Next() {
		var d datastructs.DiskInfo
		if err := rows.Scan(
			&d.Server, //потом не надо
			&d.Filesystem,
			&d.Size, //потом не надо
			&d.Used, //потом не надо
			&d.Avail,
			&d.UsePrc,
			&d.MountedOn,
			&d.Threshold,
			&d.LastCheck,
		); err != nil {
			return nil, err
		}
		disks = append(disks, d)
	}
	return disks, nil
}

func (db *Postgres) CheckRelation(ctx context.Context) error {
	var (
		dbName, tableName string
	)
	if err := db.QueryRow(ctx,
		"SELECT catalog_name FROM information_schema.schemata WHERE schema_name = 'disk_monitor'",
	).Scan(&dbName); err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			if _, err := db.Exec(ctx, "CREATE SCHEMA disk_monitor"); err != nil {
				return err
			}
			if err := db.createTable(ctx); err != nil {
				return err
			}
			return nil
		} else {
			return fmt.Errorf("unable to scan datas from information_schema.schemata: %w", err)
		}
	}

	if err := db.QueryRow(ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'disk_monitor' AND table_name = 'monitoring_disk'",
	).Scan(&tableName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := db.createTable(ctx); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unable to check information_schema.tables: %w", err)
		}
	}

	return nil
}

/*
Filesystem            Size  Used Avail Use% Mounted on
devtmpfs              7.8G     0  7.8G   0% /dev
tmpfs                 7.8G   16K  7.8G   1% /dev/shm
tmpfs                 7.8G  770M  7.1G  10% /run
tmpfs                 7.8G     0  7.8G   0% /sys/fs/cgroup
/dev/mapper/vg0-root  6.2G  3.4G  2.9G  55% /
/dev/sda1            1014M  237M  778M  24% /boot
/dev/sdb1             1.5T   30G  1.4T   3% /data
tmpfs                 1.6G     0  1.6G   0% /run/user/1000
tmpfs                 1.6G     0  1.6G   0% /run/user/0
*/

func (db *Postgres) createTable(ctx context.Context) (err error) {
	_, err = db.Exec(ctx, `CREATE TABLE disk_monitor.monitoring_disk (
		server varchar NOT NULL,
		filesystem varchar NOT NULL,
		size varchar NOT NULL,
		used varchar NOT NULL,
		available varchar NOT NULL,
		used_prc int4 NOT NULL,
		mounted_on text NOT NULL,
		threshold int4 NOT NULL,
		last_check timestamp NOT NULL DEFAULT now(),
		CONSTRAINT pk_monitoring_disk PRIMARY KEY (mounted_on) 
	)`)
	return err
}
