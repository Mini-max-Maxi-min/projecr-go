package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db        *gorm.DB
	jwtSecret = []byte(getEnv("JWT_SECRET", "supersecret"))
)

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// ---------- Models ----------

type User struct {
	ID        uint      `gorm:"primaryKey"`
	Username  string    `gorm:"uniqueIndex;size:100"`
	Password  string    `gorm:"size:255"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Exercise struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"size:100"`
	Description string    `gorm:"type:text"`
	Category    string    `gorm:"size:50"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Workout struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    *uint     `gorm:"index"`
	Title     string    `gorm:"size:150"`
	Date      *time.Time
	Comment   string    `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Exercises []WorkoutExercise `gorm:"foreignKey:WorkoutID"`
}

type WorkoutExercise struct {
	ID         uint `gorm:"primaryKey"`
	WorkoutID  uint `gorm:"index"`
	ExerciseID uint `gorm:"index"`
	Sets       int
	Reps       int
	Weight     float64
}

// ---------- Auth Helpers ----------

func hashPassword(pass string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(hash, pass string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)) == nil
}

func createToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	return token.SignedString(jwtSecret)
}

// ---------- Init DB ----------

func initDB() {
	dsn := getEnv("DATABASE_URL", "")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	if err := db.AutoMigrate(&User{}, &Exercise{}, &Workout{}, &WorkoutExercise{}); err != nil {
		log.Fatal("Migration error:", err)
	}
	seedExercises()
}

func seedExercises() {
	var count int64
	db.Model(&Exercise{}).Count(&count)
	if count == 0 {
		exs := []Exercise{
			{Name: "Push-ups", Description: "Bodyweight chest exercise", Category: "strength"},
			{Name: "Squats", Description: "Compound leg exercise", Category: "strength"},
			{Name: "Plank", Description: "Core stability", Category: "flexibility"},
			{Name: "Running - 5km", Description: "Cardio 5 kilometers run", Category: "cardio"},
			{Name: "Burpees", Description: "Full body high-intensity", Category: "cardio"},
		}
		for _, e := range exs {
			db.Create(&e)
		}
		log.Println("Seeded exercises.")
	}
}

// ---------- Telegram Bot Handlers ----------

func main() {
	initDB()

	botToken := getEnv("TELEGRAM_BOT_TOKEN", "")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bot started: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		txt := strings.TrimSpace(update.Message.Text)
		if txt == "" {
			continue
		}

		args := strings.Fields(txt)
		cmd := args[0]

		switch cmd {
		case "/start":
			msg := "🏋️ Workout Tracker Bot\n\nКоманди:\n/signup username password\n/login username password\n/exercises\n/createworkout title\n/addexercise workout_id exercise_id sets reps weight\n/myworkouts\n/stats\n/logout"
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		case "/signup":
			if len(args) < 3 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Usage: /signup username password"))
				continue
			}
			username := args[1]
			password := args[2]
			passHash, err := hashPassword(password)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Error hashing password"))
				continue
			}
			user := User{Username: username, Password: passHash}
			if err := db.Create(&user).Error; err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Error creating user (maybe exists)"))
			} else {
				token, _ := createToken(user.ID)
				resp := fmt.Sprintf("✅ Registered. Token: %s\n(You can store token client-side if needed)", token)
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, resp))
			}

		case "/login":
			if len(args) < 3 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Usage: /login username password"))
				continue
			}
			username := args[1]
			password := args[2]
			var user User
			if err := db.Where("username = ?", username).First(&user).Error; err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "User not found"))
				continue
			}
			if !checkPassword(user.Password, password) {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Wrong password"))
				continue
			}
			token, _ := createToken(user.ID)
			msg := fmt.Sprintf("✅ Logged in. Token:\n%s\n(Keep it private)", token)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		case "/exercises":
			var exs []Exercise
			db.Find(&exs)
			if len(exs) == 0 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "No exercises found"))
				continue
			}
			var sb strings.Builder
			sb.WriteString("💪 Exercises:\n")
			for _, e := range exs {
				sb.WriteString(fmt.Sprintf("%d) %s — %s (%s)\n", e.ID, e.Name, e.Description, e.Category))
			}
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, sb.String()))

		case "/createworkout":
			if len(args) < 2 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Usage: /createworkout title"))
				continue
			}
			title := strings.Join(args[1:], " ")
			now := time.Now()
			w := Workout{Title: title, Date: &now}
			db.Create(&w)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Workout '%s' created (id=%d)", title, w.ID)))

		case "/addexercise":
			// /addexercise workout_id exercise_id sets reps weight
			if len(args) < 6 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Usage: /addexercise workout_id exercise_id sets reps weight"))
				continue
			}
			wid, _ := strconv.Atoi(args[1])
			eid, _ := strconv.Atoi(args[2])
			sets, _ := strconv.Atoi(args[3])
			reps, _ := strconv.Atoi(args[4])
			weight, _ := strconv.ParseFloat(args[5], 64)
			item := WorkoutExercise{
				WorkoutID:  uint(wid),
				ExerciseID: uint(eid),
				Sets:       sets,
				Reps:       reps,
				Weight:     weight,
			}
			if err := db.Create(&item).Error; err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Error adding exercise to workout"))
			} else {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Exercise added ✅"))
			}

		case "/myworkouts":
			var workouts []Workout
			db.Preload("Exercises").Find(&workouts)
			if len(workouts) == 0 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "No workouts yet"))
				continue
			}
			var sb strings.Builder
			for _, w := range workouts {
				sb.WriteString(fmt.Sprintf("🏷 %d: %s\n", w.ID, w.Title))
				var items []WorkoutExercise
				db.Where("workout_id = ?", w.ID).Find(&items)
				for _, it := range items {
					var ex Exercise
					db.First(&ex, it.ExerciseID)
					sb.WriteString(fmt.Sprintf("   - %s: %d sets x %d reps, weight %.2f\n", ex.Name, it.Sets, it.Reps, it.Weight))
				}
				sb.WriteString("\n")
			}
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, sb.String()))

		case "/stats":
			// Simple global stats: total workouts, total exercises
			var wc int64
			db.Model(&Workout{}).Count(&wc)
			var wec int64
			db.Model(&WorkoutExercise{}).Count(&wec)
			msg := fmt.Sprintf("📊 Stats:\nTotal workouts: %d\nTotal workout-exercises: %d\n", wc, wec)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		case "/logout":
			// Bot-level: inform user to discard token
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🛑 Logout: simply discard your token client-side. (Bot does not store sessions)"))

		default:
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command. Send /start to see commands."))
		}
	}
}
