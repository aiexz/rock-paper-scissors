package middleware

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/aiexz/rock-paper-scissors/internal/game"
	"html"
	"log"
)

func MidllewareGroup(b *gotgbot.Bot, ctx *ext.Context) error {
	err := MessageLogger(b, ctx)
	if err != nil {
		return err
	}
	err = UserHandler(b, ctx)
	if err != nil {
		return err
	}
	return nil
}

func MessageLogger(b *gotgbot.Bot, ctx *ext.Context) error {

	var name string
	if ctx.EffectiveUser.Username != "" {
		name = "@" + ctx.EffectiveUser.Username
	} else {
		name = ctx.EffectiveUser.FirstName + " " + ctx.EffectiveUser.LastName
	}
	switch {
	case ctx.CallbackQuery != nil:
		log.Printf("[%s] %s", name, ctx.CallbackQuery.Data)
	case ctx.EffectiveMessage != nil:
		log.Printf("[%s] %s", name, ctx.EffectiveMessage.Text)
	}
	return nil
}

func UserHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	var user game.User
	game.DB.FirstOrCreate(&user, &game.User{UserId: ctx.EffectiveUser.Id, Name: html.EscapeString(ctx.EffectiveUser.FirstName)})
	return nil
}
