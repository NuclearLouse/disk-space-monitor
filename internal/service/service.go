package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"redits.oculeus.com/asorokin/connect/postgres"
	"redits.oculeus.com/asorokin/disk-space-monitor-src/internal/database"
	"redits.oculeus.com/asorokin/disk-space-monitor-src/internal/datastructs"
	"redits.oculeus.com/asorokin/logging"

	"github.com/AlecAivazis/survey"
	"github.com/bndr/gotabulate"
	conf "github.com/tinrab/kit/cfg"
)

var version, configFile string

func Version() {
	fmt.Println("Version =", version)
}

type Service struct {
	cfg   *config
	log   *logging.Logger
	store storer
	stop  chan struct{}
}

type config struct {
	ServerName       string           `cfg:"server_name"`
	DefaultThreshold int              `cfg:"default_threshold"`
	CheckPeriod      time.Duration    `cfg:"check_period"`
	Logger           *logging.Config  `cfg:"logger"`
	Postgres         *postgres.Config `cfg:"postgres"`
}

func New() (*Service, error) {
	c := conf.New()
	if err := c.LoadFile(configFile); err != nil {
		return nil, fmt.Errorf("load config files: %w", err)
	}
	cfg := &config{
		Logger:   logging.DefaultConfig(),
		Postgres: postgres.DefaultConfig(),
	}
	if err := c.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("mapping config files: %w", err)
	}

	return &Service{
		cfg:  cfg,
		log:  logging.New(cfg.Logger),
		stop: make(chan struct{}, 1),
	}, nil
}

func (s *Service) Start(direct bool) {
	s.log.Infof("***********************SERVICE [%s] START***********************", version)
	flog := s.log.WithField("Root", "Service")
	ctx, cancel := context.WithCancel(context.Background())
	pool, err := postgres.Connect(ctx, s.cfg.Postgres)
	if err != nil {
		flog.Fatalln("database connect:", err)
	}

	s.store = database.New(pool)

	if err := s.store.CheckRelation(ctx); err != nil {
		flog.Fatalln("check database relation:", err)
	}

	current, err := checkDisk()
	if err != nil {
		flog.Fatalln("check disk info:", err)
	}
	checkTime := time.Now()
	var forUpdate []datastructs.DiskInfo
	if !direct {
		//запуск не прямой
		fmt.Println("Current disks usage:")
		viewData(current)

		/*
			+------+---------------+---------+---------+--------------+----------+---------------+
			|   #  |   Filesystem  |   Size  |   Used  |   Available  |   Use %  |   Mounted on  |
			+======+===============+=========+=========+==============+==========+===============+
			|   1  |     C:/Git    |   466G  |   150G  |     316G     |   33 %   |       /       |
			+------+---------------+---------+---------+--------------+----------+---------------+
		*/

	CHOISE:
		for {
			ans, err := setNewThreshold(s.cfg.DefaultThreshold)
			if err != nil {
				flog.Errorln("could not get answers to interactive questions:", err)
				for _, c := range current {
					forUpdate = append(forUpdate, s.defaultTreshold(c))
				}
				break CHOISE
			}

			newThreshold := strings.TrimSuffix(ans.Valid, "%")
			values := strings.Split(ans.Disk, "|")
			mounted := values[len(values)-1]
			threshold := s.cfg.DefaultThreshold
			if !strings.Contains(newThreshold, "Default threshold") {
				threshold, err = strconv.Atoi(newThreshold)
				if err != nil {
					flog.Errorf("convert new threshold to int: %s: will be set to default: %d %%", err, s.cfg.DefaultThreshold)
					threshold = s.cfg.DefaultThreshold
				}
			}

			if threshold == 0 {
				fmt.Printf("Disk chosen: [%s] | Without warnning limit.\n", ans.Disk)
			} else {
				fmt.Printf("Disk chosen: [%s] | Set new limit: %d%%.\n", ans.Disk, threshold)
			}

			for _, c := range current {
				mountedOn := strings.Split(strings.TrimSpace(mounted), "Mounted on:")[1]
				if strings.Contains(c.MountedOn, mountedOn) {

					forUpdate = append(forUpdate, datastructs.DiskInfo{
						Filesystem: c.Filesystem,
						Size:       c.Size,
						Used:       c.Used,
						Avail:      c.Avail,
						UsePrc:     c.UsePrc,
						MountedOn:  c.MountedOn,
						Threshold:  threshold,
						LastCheck:  checkTime,
					})
				}
			}

			allComplite := false
			prompt := &survey.Confirm{
				Message: "All complite?",
			}
			survey.AskOne(prompt, &allComplite)
			if allComplite {
				break CHOISE
			}
			fmt.Scanln()
		}
	} else {

		saved, err := s.store.SavedDisk(ctx, s.cfg.ServerName)
		if err != nil {
			flog.Fatalln("get saved disk info:", err)
		}
		forUpdate = s.compare(current, saved)

	}

	if err := s.store.UpdateInfo(ctx, s.cfg.ServerName, forUpdate); err != nil {
		flog.Fatalln("update disk info:", err)
	}
	go s.worker(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGABRT)

	<-sig

	cancel()

	flog.Info("database connection closed")
	s.log.Info("***********************SERVICE STOP************************")

}

func suggestDisk(toComplete string) []string {
	infos, err := checkDisk()
	if err != nil {
		return nil
	}
	mounted := []string{}
	for j, i := range infos {
		mounted = append(mounted,
			fmt.Sprintf("Disk Num:%d | Filesystem:%s | Disk size:%s | Mounted on:%s",
				j+1,
				i.Filesystem,
				i.Size,
				cut(i.MountedOn, 50),
			))
	}
	return mounted
}

type answer struct {
	Disk  string
	Valid string
}

func setNewThreshold(defaultThreshold int) (*answer, error) {
	ans := new(answer)
	var q = []*survey.Question{
		{
			Name: "disk",
			Prompt: &survey.Input{
				Message: "Select a drive to change the limit ?",
				Suggest: suggestDisk,
				Help:    "If the limit is not set, it will be taken from the default settings. If set to '0' the disk will not be checked",
			},
			Validate: survey.Required,
		},
		{
			Name: "valid",
			Prompt: &survey.Input{
				Message: "Input new threshold:",
				Default: fmt.Sprintf("Default threshold: %d", defaultThreshold),
			},
			Validate: func(val interface{}) error {
				if lim, ok := val.(string); !ok {
					return fmt.Errorf("no value")
				} else {
					if strings.Contains(lim, "Default threshold") {
						return nil
					}
					if _, err := strconv.Atoi(lim); err != nil {
						return fmt.Errorf("non integer numeric value entered: %s", lim)
					}

				}
				return nil
			},
		},
	}
	if err := survey.Ask(q, ans); err != nil {
		return nil, err
	}
	return ans, nil
}

func viewData(data []datastructs.DiskInfo) {
	var rows [][]string
	for i, d := range data {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			d.Filesystem,
			d.Size,
			d.Used,
			d.Avail,
			fmt.Sprintf("%d %%", d.UsePrc),
			cut(d.MountedOn, 50),
		})
	}
	tab := gotabulate.Create(rows)
	tab.SetHeaders([]string{"#", "Filesystem", "Size", "Used", "Available", "Use %", "Mounted on"})
	tab.SetWrapStrings(true)
	tab.SetAlign("center")
	fmt.Println(tab.Render("grid"))
}

func cut(text string, limit int) string {
	runes := []rune(text)
	if len(runes) >= limit {
		return string(runes[:limit])
	}
	return text
}
