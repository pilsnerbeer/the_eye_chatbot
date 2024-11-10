package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"os"
)

var db *gorm.DB

type UserData struct {
	ID          uint   `gorm:"primaryKey"`
	Title       string `gorm:"not null"`
	Description string `gorm:"not null"`
	Value       string `gorm:"not null"`
}

type ApiKey struct {
	ID     uint   `gorm:"primaryKey"`
	ApiKey string `gorm:"not null"`
}

func InitDB() (*gorm.DB, error) {
	supportDir, err := getAppSupportDir()
	if err != nil {
		return nil, err
	}

	sqlitePath := supportDir + "/userdata.db"
	if _, err := os.Stat(sqlitePath); err != nil {
		file, err := os.Create(sqlitePath)
		if err != nil {
			return nil, err
		}
		file.Close()

	}

	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&UserData{}, &ApiKey{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func InsertData(db *gorm.DB, title string, desc string, value string) error {
	log.Println("Inserting data into db: ", title)
	message := UserData{Title: title, Description: desc, Value: value}
	return db.Create(&message).Error
}

func ReadDataAsJSON(db *gorm.DB) (string, error) {
	log.Println("Reading data from db")
	var data []UserData
	err := db.Find(&data).Error
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func SaveApiKey(db *gorm.DB, apiKey string) error {
	log.Println("Saving API key to db")
	err := db.Exec("DELETE FROM api_keys").Error
	if err != nil {
		return err
	}

	return db.Create(&ApiKey{ApiKey: apiKey}).Error
}

func GetApiKey(db *gorm.DB) (string, error) {
	log.Println("Getting API key from db")
	var apiKey ApiKey
	err := db.First(&apiKey).Error
	if err != nil {
		return "", err
	}
	if apiKey.ApiKey == "" {
		return "apikey", errors.New("API key not found")
	}
	return apiKey.ApiKey, nil
}

func DumpRows(db *gorm.DB) (string, error) {
	var data []UserData
	err := db.Find(&data).Error
	if err != nil {
		return "", err
	}

	var output string
	for _, row := range data {
		output += row.Title + " - " + row.Description + " - " + row.Value + "\n"
	}
	return output, nil
}

func DeleteData(db *gorm.DB) error {
	return db.Exec("DELETE FROM user_data").Error
}

func CountRows(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Model(&UserData{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetDB() (*gorm.DB, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}
	return db, nil
}
