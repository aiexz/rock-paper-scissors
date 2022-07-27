package main

import (
	"fmt"
	tele "gopkg.in/telebot.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html"
	"log"
	"os"
	"strconv"
)

const (
	ROCK     = 0
	PAPER    = 1
	SCISSORS = 2
)

func ChoiceConverter(choice string) uint8 {
	switch choice {
	case "rock":
		return ROCK
	case "paper":
		return PAPER
	case "scissors":
		return SCISSORS
	default:
		return ROCK
	}
}

type User struct {
	UserId int64 `gorm:"primary_key"`
	Name   string
}
type Game struct {
	Id            int64 `gorm:"primary_key;auto_increment;not_null"`
	Players       uint8
	Turns         []Turn  `gorm:"foreignKey:GameId;references:Id"`
	Winner        []int64 `gorm:"type:integer[]"`
	WinnerGesture uint8
}

type GestureStat struct {
	Gesture uint8
	Count   uint8
}

func (game *Game) CheckResult() {
	turns := game.Turns
	rockGesture, paperGesture, scissorsGesture := game.calculateGestures()
	if rockGesture.Count > 0 && paperGesture.Count > 0 && scissorsGesture.Count > 0 ||
		rockGesture.Count == game.Players ||
		paperGesture.Count == game.Players ||
		scissorsGesture.Count == game.Players {
		game.Winner = append(game.Winner, -1)
		game.WinnerGesture = 255
		return
	}
	var winningGesture uint8
	switch {
	case rockGesture.Count == 0:
		winningGesture = SCISSORS
	case paperGesture.Count == 0:
		winningGesture = ROCK
	case scissorsGesture.Count == 0:
		winningGesture = PAPER
	}
	for _, turn := range turns {
		if turn.Gesture == winningGesture {
			game.Winner = append(game.Winner, turn.UserId)
		}
	}
	game.WinnerGesture = winningGesture

}

func (game Game) calculateGestures() (rGesture GestureStat, pGesture GestureStat, sGesture GestureStat) {
	turns := game.Turns
	rGesture, pGesture, sGesture = GestureStat{0, 0}, GestureStat{1, 0}, GestureStat{2, 0}
	for i := 0; i < len(turns); i++ {
		switch turns[i].Gesture {
		case ROCK:
			rGesture.Count += 1
		case PAPER:
			pGesture.Count += 1
		case SCISSORS:
			sGesture.Count += 1
		}
	}
	return rGesture, pGesture, sGesture
}

type Turn struct {
	GameId  int64 `gorm:"primary_key;autoIncrement:false"`
	UserId  int64 `gorm:"primary_key;autoIncrement:false"`
	Gesture uint8
}

var db *gorm.DB

func main() {
	pref := tele.Settings{
		Token: os.Getenv("TOKEN"),
	}
	var err error
	db, err = gorm.Open(sqlite.Open("db.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatalf("cannot open an SQLite memory database: %v", err)
		return
	}
	err = db.AutoMigrate(&User{}, &Turn{}, &Game{})
	if err != nil {
		log.Fatalf("cannot migrate: %v", err)
		return
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}
	b.Use(MessageLogger)
	b.Use(UserHandler)

	b.Handle("/start", func(c tele.Context) error {
		return c.Send("Hello! I work in inline mode")
	})
	b.Handle("/help", func(c tele.Context) error {
		return c.Send("Bot follows [official rules](https://wrpsa.com/how-to-play-rock-paper-scissors-with-more-than-two-players/)")
	})
	b.Handle(tele.OnQuery, handleInlineQuery)
	b.Handle("\frock", handleTurn)
	b.Handle("\fpaper", handleTurn)
	b.Handle("\fscissors", handleTurn)
	b.Handle("\fnew_game", handleNewGame)
	log.Println("Bot is starting")
	b.Start()
}

func CheckDigits(n string) bool {
	for _, c := range n {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func handleInlineQuery(c tele.Context) error {
	var num int
	num = 2
	if CheckDigits(c.Query().Text) && c.Query().Text != "" && len(c.Query().Text) < 3 {
		var err error
		num, err = strconv.Atoi(c.Query().Text)
		if err != nil {
			return err
		}
	}
	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(keyboard.Row(keyboard.Data("Start a new game", "new_game", strconv.Itoa(num))))
	var res []tele.Result
	result := &tele.ArticleResult{Title: "Create new game", Description: fmt.Sprintf("Create new game for %d players", num),
		Text: "Start a new game!", ResultBase: tele.ResultBase{ReplyMarkup: keyboard}}
	res = append(res, result)
	res[0].SetResultID("newGame_" + strconv.Itoa(num))
	return c.Answer(&tele.QueryResponse{
		Results:   res,
		CacheTime: 3600, // a minute
	})
}

func handleNewGame(c tele.Context) error {
	num, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Send("Invalid number of players")
	}
	game := Game{Players: uint8(num)}
	db.Create(&game)
	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(keyboard.Data("Rock", "rock", strconv.Itoa(int(game.Id)))),
		keyboard.Row(keyboard.Data("Paper", "paper", strconv.Itoa(int(game.Id)))),
		keyboard.Row(keyboard.Data("Scissors", "scissors", strconv.Itoa(int(game.Id)))))
	err = c.Edit(fmt.Sprintf("New game (0/%s)", strconv.Itoa(int(game.Players))), keyboard)
	if err != nil {
		return err
	}
	return c.Respond(&tele.CallbackResponse{Text: "Game created"})

}

func handleTurn(c tele.Context) error {
	gameId, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Invalid game id"})
	}
	var choice uint8
	choice = ChoiceConverter(c.Callback().Unique)
	turn := &Turn{
		GameId:  int64(gameId),
		UserId:  c.Sender().ID,
		Gesture: choice,
	}
	res := db.FirstOrCreate(&turn)
	if res.Error != nil || res.RowsAffected == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "You already played"})
	}
	var game *Game
	err = db.Model(&Game{}).Preload("Turns").Find(&game, &Game{Id: int64(gameId)}).Error
	if err != nil {
		return err
	}
	if len(game.Turns) == int(game.Players) {
		game.CheckResult()
		db.Save(&game)
		_ = c.Respond(&tele.CallbackResponse{Text: "Game is over"})
		var winText string
		var users []*User
		var emoji string
		var userIds []int64
		for _, id := range game.Turns {
			userIds = append(userIds, id.UserId)
		}
		db.Find(&users, userIds)
		for i, turn := range game.Turns {
			status := ""
			if turn.Gesture == game.WinnerGesture {
				status = "won!"
			}
			switch turn.Gesture {
			case ROCK:
				emoji = "✊"
			case PAPER:
				emoji = "✋"
			case SCISSORS:
				emoji = "✌️️"
			}
			winText += fmt.Sprintf("%s<a href='tg://user?id=%d'>%s</a> %s\n", emoji, turn.UserId, users[i].Name, status)
		}
		if game.WinnerGesture == 255 {
			winText += "It's a draw"
		}
		return c.Edit(winText, tele.ModeHTML)

	} else {
		keyboard := &tele.ReplyMarkup{}
		keyboard.Inline(
			keyboard.Row(keyboard.Data("Rock", "rock", strconv.Itoa(int(game.Id)))),
			keyboard.Row(keyboard.Data("Paper", "paper", strconv.Itoa(int(game.Id)))),
			keyboard.Row(keyboard.Data("Scissors", "scissors", strconv.Itoa(int(game.Id)))))
		err := c.Edit(fmt.Sprintf("Waiting for other players (%d/%d)", len(game.Turns), game.Players), keyboard)
		if err != nil {
			return err
		}
		err = c.Respond(&tele.CallbackResponse{Text: "You chose " + c.Callback().Unique})
		if err != nil {
			return err
		}
		return nil
	}
}

func MessageLogger(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		var name string
		if c.Sender().Username != "" {
			name = "@" + c.Sender().Username
		} else {
			name = c.Sender().FirstName + " " + c.Sender().LastName
		}
		switch {
		case c.Callback() != nil:
			log.Printf("[%s] %s %s", name, c.Callback().Unique, c.Callback().Data)
		case c.Message() != nil:
			log.Printf("[%s] %s", name, c.Message().Text)
		}
		return next(c) // continue execution chain
	}
}

func UserHandler(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		var user User
		db.FirstOrCreate(&user, &User{UserId: c.Sender().ID, Name: html.EscapeString(c.Sender().FirstName)})
		return next(c)
	}
}
