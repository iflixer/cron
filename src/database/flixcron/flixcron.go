package flixcron

import (
	"io"
	"local/database"
	"local/database/cronLog"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Service struct {
	mu             sync.RWMutex
	dbService      *database.Service
	cronLogService *cronLog.Service
	updatePeriod   time.Duration
	crons          map[int]Cron
	cron           *cron.Cron
}

type Cron struct {
	ID          int
	Name        string
	Expression  string
	Method      string
	TargetUrl   string
	TargetHost  string
	LogResponse bool
	Timeout     int
	LastFired   *time.Time
	UpdatedAt   *time.Time
	CronEntryID cron.EntryID
	IsRunning   bool
}

func (c *Cron) TableName() string {
	return os.Getenv("CRON_TABLE")
}

func (s *Service) Start() {
	s.cron.Start()

}

func (s *Service) Stop() {
	s.cron.Stop()
}

func (s *Service) Export() (res []*Cron) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.crons {
		cc := c
		res = append(res, &cc)
	}
	return
}

func NewService(dbService *database.Service, cronLogService *cronLog.Service, updatePeriod int) (s *Service, err error) {

	s = &Service{
		dbService:      dbService,
		updatePeriod:   time.Duration(updatePeriod),
		cronLogService: cronLogService,
		crons:          make(map[int]Cron),
		cron:           cron.New(),
	}

	err = s.loadData()

	go s.loadWorker()

	return
}

func (s *Service) loadWorker() {
	for {
		time.Sleep(time.Second * s.updatePeriod)
		if err := s.loadData(); err != nil {
			log.Println(err)
		}
	}
}

func (s *Service) execJob(cronId int) {
	log.Println("exec job", cronId)
	if c, ok := s.crons[cronId]; ok {
		if c.IsRunning {
			log.Printf("job is already running, id: %d, name: %s\n", cronId, c.Name)
			return
		}
		c.IsRunning = true
		s.crons[cronId] = c
		defer func() {
			c.IsRunning = false
			s.crons[cronId] = c
		}()
		start := time.Now()
		logId := s.cronLogService.Log(0, cronId, 0, "", 0)
		req, err := http.NewRequest(c.Method, c.TargetUrl, nil)
		if err != nil {
			log.Printf("could not create request, id: %d, name: %s, error: %s\n", cronId, c.Name, err.Error())
		}
		if c.TargetHost != "" {
			req.Host = c.TargetHost
		}
		//req.Header.Set("Content-Type", "application/json")

		client := http.Client{
			Timeout: time.Duration(c.Timeout) * time.Second,
		}

		res, err := client.Do(req)
		if err != nil {
			log.Printf("error making http request, id: %d, name: %s, error: %s\n", cronId, c.Name, err.Error())
			s.cronLogService.Log(logId, cronId, 408, err.Error(), int(time.Since(start).Seconds()))
			return
		}
		defer res.Body.Close()

		respString := ""
		if c.LogResponse {
			respBytes, _ := io.ReadAll(res.Body)
			respString = string(respBytes)
		}
		s.cronLogService.Log(logId, cronId, res.StatusCode, respString, int(time.Since(start).Seconds()))
	}
}

func (s *Service) loadData() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var newCrons []*Cron
	if err = s.dbService.DB.Where("active=?", 1).Find(&newCrons).Error; err == nil {
		// update and create
		for _, newCron := range newCrons {
			oldCron, found := s.crons[newCron.ID]
			if found && oldCron.UpdatedAt.Equal(*newCron.UpdatedAt) {
				continue
			}
			if oldCron.CronEntryID > 0 {
				s.cron.Remove(oldCron.CronEntryID)
			}
			//	c.AddFunc("0 30 * * * *", func() { fmt.Println("Every hour on the half hour") })
			var err1 error
			newCron.CronEntryID, err1 = s.cron.AddFunc(newCron.Expression, func() { s.execJob(newCron.ID) })
			if err1 != nil {
				log.Printf("error AddFunc, id: %d, name: %s, error: %s\n", newCron.ID, newCron.Name, err1.Error())
			} else {
				s.crons[newCron.ID] = *newCron
				if found {
					log.Printf("cron updated, id: %d, name: %s\n", newCron.ID, newCron.Name)
				} else {
					log.Printf("cron added, id: %d, name: %s\n", newCron.ID, newCron.Name)
				}
			}

		}
		// remove
		for _, oldCron := range s.crons {
			found := false
			for _, newCron := range newCrons {
				if oldCron.ID == newCron.ID {
					found = true
					break
				}
			}
			if !found {
				if oldCron.CronEntryID > 0 {
					s.cron.Remove(oldCron.CronEntryID)
				}
				delete(s.crons, oldCron.ID)
				log.Printf("cron removed, id: %d, name: %s\n", oldCron.ID, oldCron.Name)
			}
		}
	}
	return
}
