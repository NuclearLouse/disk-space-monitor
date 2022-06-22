package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"redits.oculeus.com/asorokin/disk-space-monitor-src/internal/datastructs"
)

func (s *Service) worker(ctx context.Context) {
	flog := s.log.WithField("Root", "Check Worker")
	flog.Info("start worker")
	tick := time.NewTicker(s.cfg.CheckPeriod)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.C:

			current, err := checkDisk()
			if err != nil {
				flog.Errorln("check current disk info:", err)
			}

			saved, err := s.store.SavedDisk(ctx, s.cfg.ServerName)
			if err != nil {
				flog.Errorln("get saved disk info:", err)
			}

			info := s.compare(current, saved)
			var exceeds []datastructs.DiskInfo
			var warn bool
			buf := &bytes.Buffer{}
			for _, i := range info {
				if i.Threshold == 0 {
					continue
				}
				if i.UsePrc >= i.Threshold {
					warn = true
					exceeds = append(exceeds, i)
				}

			}
			if warn {
				if err := json.NewEncoder(buf).Encode(exceeds); err != nil {
					flog.Errorln("encode disk info to json:", err)
				}
				flog.Warnf("threshold exceeded! %s", buf.String())
			}
			if err := s.store.UpdateInfo(ctx, s.cfg.ServerName, info); err != nil {
				flog.Errorln("update last checked info:", err)
			}
		default:
			time.Sleep(500 * time.Millisecond)
		}

	}
}

func checkDisk() ([]datastructs.DiskInfo, error) {
	cmd := exec.Command("sh", "-c", "df -h")
	output, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer output.Close()

	cmd.Start()
	chechTime := time.Now()

	scanner := bufio.NewScanner(output)
	scanner.Split(bufio.ScanLines)

	var (
		i     int
		infos []datastructs.DiskInfo
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

			usePrc, err := strconv.Atoi(strings.Split(newRow[4], "%")[0])
			if err != nil {
				return nil, fmt.Errorf("convert used prc to int: %w", err)
			}
			infos = append(infos, datastructs.DiskInfo{
				Filesystem: newRow[0],
				Size:       newRow[1],
				Used:       newRow[2],
				Avail:      newRow[3],
				UsePrc:     usePrc,
				MountedOn:  newRow[5],
				LastCheck:  chechTime,
			})
		}
		i++
	}

	cmd.Wait()
	return infos, nil
}

func (s *Service) defaultTreshold(data datastructs.DiskInfo) datastructs.DiskInfo {
	return datastructs.DiskInfo{
		Filesystem: data.Filesystem,
		Size:       data.Size,
		Used:       data.Used,
		Avail:      data.Avail,
		UsePrc:     data.UsePrc,
		MountedOn:  data.MountedOn,
		Threshold:  s.cfg.DefaultThreshold,
		LastCheck:  data.LastCheck,
	}
}

func (s *Service) compare(current, saved []datastructs.DiskInfo) map[string]datastructs.DiskInfo {

	diff := make(map[string]datastructs.DiskInfo)
	if len(saved) == 0 {
		for _, c := range current {
			diff[c.MountedOn] = s.defaultTreshold(c)
		}
		return diff
	}

	for _, c := range current {
		var contain bool
		for _, sd := range saved {
			if c.MountedOn == sd.MountedOn {
				contain = true
				diff[c.MountedOn] = datastructs.DiskInfo{
					Filesystem: sd.Filesystem,
					Size:       sd.Size,
					Used:       c.Used,
					Avail:      c.Avail,
					UsePrc:     c.UsePrc,
					MountedOn:  sd.MountedOn,
					Threshold:  sd.Threshold,
					LastCheck:  c.LastCheck,
				}
				break
			}
		}
		if !contain {
			diff[c.MountedOn] = s.defaultTreshold(c)
		}
	}
	return diff
}
