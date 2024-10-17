package cronLog

import (
	"local/database"
	"os"
)

type Service struct {
	dbService *database.Service
}

type CronLog struct {
	ID           int
	CronId       int
	HttpCode     int
	HttpResponse string
	Seconds      int
}

func (c *CronLog) TableName() string {
	return os.Getenv("CRON_LOG_TABLE")
}

func (s *Service) Log(cronId, httpCode int, httpResponse string, seconds int) {
	l := CronLog{CronId: cronId, HttpCode: httpCode, HttpResponse: httpResponse, Seconds: seconds}
	s.dbService.DB.Create(&l)
}

func NewService(dbService *database.Service) (s *Service, err error) {

	s = &Service{
		dbService: dbService,
	}

	return
}
