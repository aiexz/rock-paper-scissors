package main

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/inlinequery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/aiexz/rock-paper-scissors/internal/bot"
	"github.com/aiexz/rock-paper-scissors/internal/game"
	"github.com/aiexz/rock-paper-scissors/internal/middleware"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	token, ok := os.LookupEnv("TOKEN")
	if !ok {
		log.Fatal("No bot token is set")
	}
	var err error
	db, err := gorm.Open(sqlite.Open("db.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatalf("cannot open an SQLite memory database: %v", err)
		return
	}
	game.DB = db
	err = db.AutoMigrate(&game.User{}, &game.Turn{}, &game.Game{})
	if err != nil {
		log.Fatalf("cannot migrate: %v", err)
		return
	}

	b, err := gotgbot.NewBot(token, &gotgbot.BotOpts{
		UseTestEnvironment: false,
		Client:             http.Client{},
		DefaultRequestOpts: &gotgbot.RequestOpts{
			Timeout: gotgbot.DefaultTimeout,
			APIURL:  gotgbot.DefaultAPIURL,
		},
	})
	if err != nil {
		log.Fatal(err)
		return
	}
	updater := ext.NewUpdater(&ext.UpdaterOpts{
		ErrorLog: nil,
		DispatcherOpts: ext.DispatcherOpts{
			// If an error is returned by a handler, log it and continue going.
			Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
				fmt.Println("an error occurred while handling update:", err.Error())
				return ext.DispatcherActionNoop
			},
			MaxRoutines: ext.DefaultMaxRoutines,
		},
	})
	dispatcher := updater.Dispatcher
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.All, middleware.MidllewareGroup), 0)
	dispatcher.AddHandlerToGroup(handlers.NewInlineQuery(inlinequery.All, middleware.MidllewareGroup), 0)
	dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.All, middleware.MidllewareGroup), 0)
	dispatcher.AddHandlerToGroup(handlers.NewCommand("start", bot.HandleStart), 1)
	dispatcher.AddHandlerToGroup(handlers.NewCommand("help", bot.HandleStart), 1)
	dispatcher.AddHandlerToGroup(handlers.NewInlineQuery(inlinequery.All, bot.HandleInlineQuery), 1)
	dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("rock"), bot.HandleTurn), 1)
	dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("paper"), bot.HandleTurn), 1)
	dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("scissors"), bot.HandleTurn), 1)
	dispatcher.AddHandlerToGroup(handlers.NewCallback(callbackquery.Prefix("new_game"), bot.HandleNewGame), 1)
	log.Println("Bot is starting")
	webhookAddress, ok := os.LookupEnv("URL")
	if ok {
		webhookOpts := ext.WebhookOpts{
			Listen:      "0.0.0.0", // This example assumes you're in a dev environment running ngrok on 8080.
			Port:        8080,
			URLPath:     token,    // Using a secret (like the token) as the endpoint ensure that strangers aren't crafting fake updates.
			SecretToken: "secret", // Setting a webhook secret (must be here AND in SetWebhook!) ensures that the webhook is set by you.
		}
		err = updater.StartWebhook(b, webhookOpts)
		if err != nil {
			panic("failed to start webhook: " + err.Error())
		}

		// Get the full webhook URL that we are expecting to receive updates at.
		webhookURL := webhookOpts.GetWebhookURL(webhookAddress)

		// Tell telegram where they should send updates for you to receive them in a secure manner.
		_, err = b.SetWebhook(webhookURL, &gotgbot.SetWebhookOpts{
			MaxConnections:     100,
			DropPendingUpdates: true,
			SecretToken:        "secret", // The secret token passed at webhook start time.
		})

	} else {
		err = updater.StartPolling(b, &ext.PollingOpts{
			DropPendingUpdates: true,
			GetUpdatesOpts: gotgbot.GetUpdatesOpts{
				Timeout: 9,
				RequestOpts: &gotgbot.RequestOpts{
					Timeout: time.Second * 60,
				},
			},
		})
	}
	if err != nil {
		panic("failed to start: " + err.Error())
	}

	fmt.Printf("%s has been started...\n", b.User.Username)
	updater.Idle()

}
