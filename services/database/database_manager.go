package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DatabaseManager struct {
	appDB  *gorm.DB
	dbsDir string
}

func NewDatabaseManager() *DatabaseManager {
	db, err := gorm.Open(sqlite.Open("dbs/master.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to load db: ", err)
	}
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA foreign_keys=ON")
	db.Exec("PRAGMA busy_timeout=5000")

	return &DatabaseManager{
		appDB:      db,
		dbsDir:        "./dbs",
	}
}

func (dm *DatabaseManager) App() *gorm.DB {
	return dm.appDB
}
