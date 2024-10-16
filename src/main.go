package main

import (
	"local/database"
	"local/database/cronLog"
	"local/database/flixcron"
	"local/server"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("START")

	log.Println("runtime.GOMAXPROCS:", runtime.GOMAXPROCS(0))

	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Cant load .env: ", err)
	}

	port := os.Getenv("HTTP_PORT")

	mysqlURL := os.Getenv("MYSQL_URL")
	if os.Getenv("MYSQL_URL_FILE") != "" {
		mysqlURL_, err := os.ReadFile(os.Getenv("MYSQL_URL_FILE"))
		if err != nil {
			log.Fatal(err)
		}
		mysqlURL = strings.TrimSpace(string(mysqlURL_))
	}

	dbService, err := database.NewService(mysqlURL)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("dbService OK")
	}

	cronLogService, err := cronLog.NewService(dbService)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("cronLogService OK")
	}

	cronService, err := flixcron.NewService(dbService, cronLogService, 60)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("cronService OK")
	}
	cronService.Start()

	serverService, err := server.NewService(port, cronService)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("starting server...")
	serverService.Run()

}
