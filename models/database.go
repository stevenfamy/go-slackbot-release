package models

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stevenfamy/go-slackbot-release/config"
)

var DB *sql.DB

type dbConfig struct {
	dbHost     string
	dbPort     string
	dbName     string
	dbUser     string
	dbPassword string
}

func ConnectDatabase() {
	var config = dbConfig{
		config.GetConfig("MYSQL_HOST"),
		config.GetConfig("MYSQL_PORT"),
		config.GetConfig("MYSQL_DB"),
		config.GetConfig("MYSQ_USER"),
		config.GetConfig("MYSQL_PASSWORD"),
	}

	connectionString := config.dbUser + ":" + config.dbPassword + "@tcp(" + config.dbHost + ":" + config.dbPort + ")/" + config.dbName

	db, err := sql.Open("mysql", connectionString)

	if err != nil {
		panic(err.Error())
	}

	DB = db
}
