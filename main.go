package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/allen-b1/territ-v3/bot"
)

var userId string
var userToken string
var server string
var roomId string
var private bool
var type_ string
var name string

func init() {
	flag.StringVar(&userId, "id", "", "user id")
	flag.StringVar(&userToken, "token", "", "user token")
	flag.StringVar(&server, "server", "na", "generals.io server (na, eu, bot)")
	flag.StringVar(&type_, "type", "custom", "type of game")
	flag.StringVar(&roomId, "room", "", "custom room id ('' for FFA)")
	flag.StringVar(&name, "name", "", "username")
	flag.BoolVar(&private, "private", false, "private game")
}

func main() {
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

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

	if name != "" {
		bt.SetUsername(name)
	}

	domain := map[bot.Server]string{
		bot.NA:  "http://generals.io",
		bot.BOT: "http://bot.generals.io",
		bot.EU:  "http://eu.generals.io",
	}[srv]

	if type_ == "ffa" {
		err = bt.JoinFFA()
		fmt.Println(domain + "/?queue=main")
	} else if type_ == "2v2" {
		err = bt.Join2v2(roomId)
		fmt.Println(domain + "/teams/" + roomId)
	} else if type_ == "1v1" {
		err = bt.Join1v1()
		fmt.Println(domain + "/?queue=1v1")
	} else {
		err = bt.JoinCustom(roomId, private)
		fmt.Println(domain + "/games/" + roomId)
	}

	if err != nil {
		panic(err)
	}

	err = bt.Listen()
	if err != nil {
		log.Println(err)
	}
}
