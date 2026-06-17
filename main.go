package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Record struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Score    int    `json:"score"`
}

func main() {
	godotenv.Load()
	token := os.Getenv("BOT_TOKEN")
	webAppURL := os.Getenv("WEB_APP_URL")

	if token == "" {
		log.Fatal("❌ BOT_TOKEN не найден в .env файле!")
	}

	// База данных
	var err error
	db, err = sql.Open("sqlite3", "records.db")
	if err != nil {
		log.Fatal(err)
	}
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS records (
		user_id INTEGER PRIMARY KEY,
		username TEXT,
		score INTEGER,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Telegram Bot
	b, err := bot.New(token, bot.WithDefaultHandler(defaultHandler))
	if err != nil {
		log.Fatal(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, startHandler(webAppURL))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/leaderboard", bot.MatchTypeExact, leaderboardHandler)

	fmt.Println("🚀 Бот успешно запущен!")
	fmt.Println("🌐 Mini App: http://localhost:8080/static/index.html")

	go b.Start(context.Background())

	// Web сервер
	r := gin.Default()
	r.POST("/submit-score", submitScore)
	r.Static("/static", "./web")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/static/index.html")
	})

	log.Fatal(r.Run(":8080"))
}

func startHandler(webAppURL string) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}
		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "🎮 Играть", WebApp: &models.WebAppInfo{URL: webAppURL + "/static/index.html"}}},
			},
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "🔞 <b>18+ Игра «Пирсинг или нет?»</b>\n\n60 секунд на максимум правильных ответов!",
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: kb,
		})
	}
}

func leaderboardHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	rows, _ := db.Query("SELECT username, score FROM records ORDER BY score DESC LIMIT 10")
	defer rows.Close()

	text := "🏆 <b>Топ игроков</b>\n\n"
	i := 1
	for rows.Next() {
		var username string
		var score int
		rows.Scan(&username, &score)
		text += fmt.Sprintf("%d. %s — %d очков\n", i, username, score)
		i++
	}
	if i == 1 {
		text += "Пока нет рекордов."
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Команды:\n/start — играть\n/leaderboard — таблица рекордов",
		})
	}
}

func submitScore(c *gin.Context) {
	var rec Record
	if err := c.BindJSON(&rec); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("INSERT OR REPLACE INTO records (user_id, username, score) VALUES (?, ?, ?)",
		rec.UserID, rec.Username, rec.Score)

	if err == nil {
		c.JSON(200, gin.H{"status": "ok"})
	} else {
		c.JSON(500, gin.H{"error": err.Error()})
	}
}