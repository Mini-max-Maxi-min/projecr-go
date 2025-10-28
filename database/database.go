package database

import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "workout-tracker/models"
)

var DB *gorm.DB

func ConnectDB() {
    db, err := gorm.Open(sqlite.Open("workout.db"), &gorm.Config{})
    if err != nil {
        panic("Не вдалося підключитися до бази даних")
    }

    db.AutoMigrate(&models.User{}, &models.Exercise{}, &models.Workout{})
    DB = db
}
