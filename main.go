package main

import (
	"flag"
	"github.com/allen-b1/territ-v3/bot"
)

var userId string
var userToken string
var server string
var roomId string
var private bool

func init() {
	flag.StringVar(&userId, "id", "", "user id")
	flag.StringVar(&userToken, "token", "", "user token")
	flag.StringVar(&server, "server", "na", "generals.io server (na, eu, bot)")
	flag.StringVar(&roomId, "room", "imabot", "custom room id ('' for FFA)")
	flag.BoolVar(&private, "private", false, "private game")
}

func main() {
	flag.Parse()

	srv := bot.NA
	if server == "bot" {
		srv = bot.BOT
	}
	if server == "eu" {
		srv = bot.EU
	}

	if userId == "" {
		panic("user id not specified")
	}
	if userToken == "" && srv != bot.BOT {
		panic("user token not specified")
	}

	bt, err := bot.New(srv, userId, userToken)
	if err != nil {
		panic(err)
	}

	if roomId != "" {
		err = bt.JoinCustom(roomId, private)
	} else {
		err = bt.JoinFFA()
	}
	if err != nil {
		panic(err)
	}

	err = bt.Listen()
	if err != nil {
		panic(err)
	}
}
