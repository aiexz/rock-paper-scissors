package bot

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/aiexz/rock-paper-scissors/internal/game"
	"log"
	"math/rand"
	"strconv"
	"strings"
)

const (
	ROCK     = 0
	PAPER    = 1
	SCISSORS = 2
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

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
func CheckDigits(n string) bool {
	for _, c := range n {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func HandleInlineQuery(b *gotgbot.Bot, ctx *ext.Context) error {
	var num int
	num = 2
	if ctx.InlineQuery.Query != "" && len(ctx.InlineQuery.Query) < 3 && CheckDigits(ctx.InlineQuery.Query) {
		var err error
		num, err = strconv.Atoi(ctx.InlineQuery.Query)
		if err != nil {
			return err
		}
	}
	var res []gotgbot.InlineQueryResult
	id := make([]byte, 8)
	for i := range id {
		id[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}

	result := &gotgbot.InlineQueryResultArticle{Id: string(id), Title: "Create new game", Description: fmt.Sprintf("Create new game for %d players", num),
		InputMessageContent: gotgbot.InputTextMessageContent{MessageText: "Starting new game..."},
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{
				{Text: "Start a new game", CallbackData: "new_game|" + strconv.Itoa(num)},
			}},
		}}
	res = append(res, result)
	_, err := ctx.InlineQuery.Answer(b, res, &gotgbot.AnswerInlineQueryOpts{CacheTime: 0})
	if err != nil {
		return err
	}
	return nil
}

func HandleNewGame(b *gotgbot.Bot, ctx *ext.Context) error {
	num, err := strconv.Atoi(strings.Split(ctx.CallbackQuery.Data, "|")[1])
	if err != nil {
		_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "Invalid number of players"})
		if err != nil {
			return err
		}
		return nil
	}
	currentGame := game.Game{Players: uint8(num)}
	currentGame.Create()
	gameIdString := strconv.Itoa(int(currentGame.Id))
	MARKUP := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: "Rock", CallbackData: "rock|" + gameIdString}},
			{{Text: "Paper", CallbackData: "paper|" + gameIdString}},
			{{Text: "Scissors", CallbackData: "scissors|" + gameIdString}},
		},
	}
	_, _, err = b.EditMessageText("New Game (0/"+strconv.Itoa(int(currentGame.Players))+")", &gotgbot.EditMessageTextOpts{InlineMessageId: ctx.CallbackQuery.InlineMessageId, ReplyMarkup: MARKUP})
	if err != nil {
		return err
	}
	_, err = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "New game created"})
	if err != nil {
		return err
	}
	return nil

}

func HandleTurn(b *gotgbot.Bot, ctx *ext.Context) error {
	gameId, err := strconv.Atoi(strings.Split(ctx.CallbackQuery.Data, "|")[1])
	if err != nil {
		_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "Invalid game id"})
		if err != nil {
			return err
		}
		return nil
	}
	var choice uint8
	uniqueData := strings.Split(ctx.CallbackQuery.Data, "|")[0] //if we get an error we just ignore it
	choice = ChoiceConverter(uniqueData)
	turn := &game.Turn{
		GameId:  int64(gameId),
		UserId:  ctx.CallbackQuery.From.Id,
		Gesture: choice,
	}
	res := game.DB.FirstOrCreate(&turn)
	if res.Error != nil || res.RowsAffected == 0 {
		_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "You already played"})
		if err != nil {
			return err
		}
		return nil
	}
	var currentGame *game.Game
	res = game.DB.Model(&game.Game{}).Preload("Turns").Find(&currentGame, &game.Game{Id: int64(gameId)})
	if res.Error != nil {
		log.Printf("Error finding game: %s", res.Error)
		return err
	}
	if len(currentGame.Turns) == int(currentGame.Players) {
		currentGame.CheckResult()
		currentGame.Save()
		_, err := ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "Game is over"})
		if err != nil {
			return err
		}
		var winText string
		var users []*game.User
		var emoji string
		var userIds []int64
		for _, id := range currentGame.Turns {
			userIds = append(userIds, id.UserId)
		}
		game.DB.Find(&users, userIds)
		for i, turn := range currentGame.Turns {
			status := ""
			if turn.Gesture == currentGame.WinnerGesture {
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
		if currentGame.WinnerGesture == 255 {
			winText += "It's a draw"
		}
		_, _, err = b.EditMessageText(winText, &gotgbot.EditMessageTextOpts{InlineMessageId: ctx.CallbackQuery.InlineMessageId, ParseMode: "HTML"})
		if err != nil {
			return err
		}
		return nil

	} else {
		gameNumberString := strconv.Itoa(int(currentGame.Id))
		line := fmt.Sprintf("Waiting for other players (%d/%d)", len(currentGame.Turns), currentGame.Players)
		_, _, err := b.EditMessageText(line, &gotgbot.EditMessageTextOpts{
			InlineMessageId: ctx.CallbackQuery.InlineMessageId,
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
					{{Text: "Rock", CallbackData: "rock|" + gameNumberString}},
					{{Text: "Paper", CallbackData: "paper|" + gameNumberString}},
					{{Text: "Scissors", CallbackData: "scissors|" + gameNumberString}},
				},
			},
		})
		if err != nil {
			return err
		}
		_, err = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "You chose " + uniqueData})
		if err != nil {
			return err
		}
		return nil
	}
}

func HandleStart(b *gotgbot.Bot, ctx *ext.Context) error {
	_, err := ctx.EffectiveMessage.Reply(b, "Welcome to Rock Paper Scissors bot! Click the button below to start a game", &gotgbot.SendMessageOpts{
		ParseMode: "html",
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{
				{Text: "Start a game", SwitchInlineQuery: " "},
			}},
		},
	})
	if err != nil {
		return err
	}
	return nil
}
