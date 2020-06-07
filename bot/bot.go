package bot

import (
	"bitbucket.org/allenb123/socketio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type Server string

const (
	NA  Server = "ws.generals.io"
	EU         = "euws.generals.io"
	BOT        = "botws.generals.io"
)

type roomInfo struct {
	teams map[string]int
}

type Bot struct {
	cl *socketio.Client

	server Server
	id     string
	token  string

	initmsg string

	username string
	number   int

	// map chat => trivia :)
	trivia map[string]*Trivia

	// Information about the room
	// No way to tell *which* room, unfortunately :(
	room roomInfo

	// Information about the game
	state *state
	turn  int
}

func New(server Server, id string, token string) (*Bot, error) {
	bt := new(Bot)
	bt.id = id
	bt.token = token
	bt.server = server
	bt.initmsg = "Hi! #" + fmt.Sprint(time.Now().UnixNano()%10000)
	bt.number = 1

	bt.room = roomInfo{}
	bt.room.teams = make(map[string]int)

	bt.trivia = make(map[string]*Trivia)

	var err error
	bt.cl, err = socketio.New(string(server), false)
	if err != nil {
		return nil, err
	}

	bt.initHandlers()

	return bt, nil
}

func (bt *Bot) initHandlers() {
	bt.cl.On("chat_message", func(data ...interface{}) {
		chatroom := data[0].(string)
		m := data[1].(map[string]interface{})
		text := fmt.Sprint(m["text"])
		author := fmt.Sprint(m["username"])

		if text == bt.initmsg && bt.username == "" {
			bt.username = author

			last := bt.username[len(bt.username)-1]
			if last >= '0' && last <= '9' {
				bt.number = int(last - byte('0'))
			}
			log.Println("You are: " + bt.username + " (" + fmt.Sprint(bt.number) + ")")
		}

		if author == bt.username {
			return
		}

		if text == "force" || text == "go" {
			bt.cl.Emit("chat_message", chatroom, "Type '/force' to force start")
		}

		if !strings.HasPrefix(text, fmt.Sprint(bt.number)+"/") && !strings.HasPrefix(text, "/") {
			return
		}

		fields := strings.Fields(text)

		if strings.HasPrefix(text, "/") {
			fields[0] = fields[0][1:]
		} else {
			fields[0] = fields[0][2:]
		}

		room := ""
		if strings.HasPrefix(chatroom, "chat_custom_queue_") {
			room = chatroom[len("chat_custom_queue_"):]
		}

		switch fields[0] {
		case "echo":
			bt.cl.Emit("chat_message", chatroom, strings.Join(fields[1:], " "))
		case "whoami":
			bt.cl.Emit("chat_message", chatroom, fmt.Sprint(m["username"]))
		case "pwd":
			if room != "" {
				bt.cl.Emit("chat_message", chatroom, "/games/"+room)
			}
		case "bash":
			bt.cl.Emit("chat_message", chatroom, "Nice try.")

		case "force":
			bt.cl.Emit("set_force_start", room, true)
		case "speed":
			speed := "2"
			if len(fields) >= 2 {
				speed = fields[1]
			}
			bt.cl.Emit("set_custom_options", room, map[string]interface{}{"game_speed": speed})
		case "map":
			bt.cl.Emit("set_custom_options", room, map[string]interface{}{"map": strings.Join(fields[1:], " ")})
		case "maplist":
			list := "top"
			if len(fields) >= 2 {
				list = strings.ToLower(fields[1])
			}
			resp, err := http.Get("http://generals.io/api/maps/lists/" + list)
			if err != nil {
				log.Println(err)
				break
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
			}
			out := make([]map[string]interface{}, 0)
			json.Unmarshal(body, &out)
			if len(out) != 0 {
				map_ := fmt.Sprint(out[rand.Intn(len(out))]["title"])
				bt.cl.Emit("set_custom_options", room, map[string]interface{}{"map": map_})
			}
		case "empty":
			size := "0.1"
			if len(fields) >= 2 {
				size = fields[1]
			}
			bt.cl.Emit("set_custom_options", room, map[string]interface{}{
				"map":              "",
				"city_density":     0,
				"mountain_density": 0,
				"swamp_density":    0,
				"width":            size,
				"height":           size,
			})
		case "team":
			bt.cl.Emit("set_custom_team", room, bt.room.teams[author])
		case "help":
			if bt.number == 1 {
				go func() {
					msgs := []string{
						"Incomplete list of commands:",
						"* /force",
						"* /speed [1|2|3|4]",
						"* /map MAP",
						"* /team",
						"Source code: https://github.com/allen-b1/territ-v3",
					}

					for _, msg := range msgs {
						bt.cl.Emit("chat_message", chatroom, msg)
						time.Sleep(500 * time.Millisecond)
					}
				}()
			}

		case "surrender":
			if bt.turn >= 1000 {
				bt.cl.Disconnect()
			} else {
				bt.cl.Emit("chat_message", chatroom, "No.")
			}
		case "trivia":
			if len(fields) < 2 {
				go func() {
					msgs := []string{
						"* /trivia start",
						"* /trivia question",
						"* /trivia guess [GUESS]",
						"* /trivia scores",
						"Questions taken from the Open Trivia Database",
					}

					for _, msg := range msgs {
						bt.cl.Emit("chat_message", chatroom, msg)
						time.Sleep(500 * time.Millisecond)
					}
				}()
				break
			}
			switch fields[1] {
			case "start":
				if bt.trivia[chatroom] == nil {
					bt.trivia[chatroom] = NewTrivia()
					t := bt.trivia[chatroom]
					question, points := t.Question()
					bt.cl.Emit("chat_message", chatroom, "Let's get started! No Googling (or Binging or Yahooing or DuckDuckGoing) allowed.")
					time.Sleep(500 * time.Millisecond)
					bt.cl.Emit("chat_message", chatroom, "First question: "+question+" [+"+fmt.Sprint(points)+"]")
				} else {
					bt.cl.Emit("chat_message", chatroom, "Trivia game already started.")
				}
			case "question":
				t := bt.trivia[chatroom]
				if t == nil {
					bt.cl.Emit("chat_message", chatroom, "Trivia game hasn't started yet. Type '/trivia start' to start.")
					return
				}
				question, points := t.Question()
				bt.cl.Emit("chat_message", chatroom, "Current question: "+question+" [+"+fmt.Sprint(points)+"]")
			case "skip":
				t := bt.trivia[chatroom]
				if t == nil {
					bt.cl.Emit("chat_message", chatroom, "Trivia game hasn't started yet. Type '/trivia start' to start.")
					return
				}
				answer, err := t.Skip()
				if err != nil {
					bt.cl.Emit("chat_message", chatroom, err.Error())
				} else {
					question, points := t.Question()
					bt.cl.Emit("chat_message", chatroom, "Answer was: "+answer)
					time.Sleep(500 * time.Millisecond)
					bt.cl.Emit("chat_message", chatroom, "Next question: "+question+" [+"+fmt.Sprint(points)+"]")
				}
			case "guess":
				t := bt.trivia[chatroom]
				if t == nil {
					bt.cl.Emit("chat_message", chatroom, "Trivia game hasn't started yet. Type '/trivia start' to start.")
					return
				}
				_, points := t.Question()
				answer := t.Guess(strings.Join(fields[2:], " "), author)
				if answer != "" {
					bt.cl.Emit("chat_message", chatroom, "Correct! Answer was: "+answer)
					scores, order := t.Scores()
					go func() {
						time.Sleep(500 * time.Millisecond)
						for _, player := range order {
							if player == author {
								bt.cl.Emit("chat_message", chatroom, player+": "+fmt.Sprint(scores[player])+" (+"+fmt.Sprint(points)+")")
							} else {
								bt.cl.Emit("chat_message", chatroom, player+": "+fmt.Sprint(scores[player]))
							}
							time.Sleep(500 * time.Millisecond)
						}
						question, points := t.Question()
						bt.cl.Emit("chat_message", chatroom, "Next question: "+question+" [+"+fmt.Sprint(points)+"]")
					}()
				} else {
					bt.cl.Emit("chat_message", chatroom, "Wrong!")
				}
			case "scores":
				t := bt.trivia[chatroom]
				if t == nil {
					bt.cl.Emit("chat_message", chatroom, "Trivia game hasn't started yet. Type '/trivia start' to start.")
					return
				}
				scores, order := t.Scores()
				go func() {
					for _, player := range order {
						bt.cl.Emit("chat_message", chatroom, player+": "+fmt.Sprint(scores[player]))
						time.Sleep(500 * time.Millisecond)
					}
				}()
			}
		}
	})

	bt.cl.On("queue_update", func(data ...interface{}) {
		m := data[0].(map[string]interface{})

		players := m["usernames"].([]interface{})
		teams := m["teams"].([]interface{})
		for i := 0; i < len(players); i++ {
			bt.room.teams[fmt.Sprint(players[i])] = int(teams[i].(float64))
		}
	})

	bt.cl.On("game_start", func(data ...interface{}) {
		log.Println("game started")
		m := data[0].(map[string]interface{})

		playerIndex := int(m["playerIndex"].(float64))
		teamsMap := make(map[int]bool)
		teams := m["teams"].([]interface{})
		for player, team := range teams {
			if team == teams[playerIndex] {
				teamsMap[player] = true
			}
		}
		bt.state = newState(playerIndex, teamsMap)

		rawSwamps := m["swamps"].([]interface{})
		swamps := make(map[int]bool)
		for _, swamp := range rawSwamps {
			swamps[int(swamp.(float64))] = true
		}
		bt.state.init(swamps)
	})

	bt.cl.On("game_update", func(data ...interface{}) {
		m := data[0].(map[string]interface{})
		rawMapDiff := m["map_diff"].([]interface{})
		rawCitiesDiff := m["cities_diff"].([]interface{})
		rawGenerals := m["generals"].([]interface{})
		mapDiff := make([]int, len(rawMapDiff))
		citiesDiff := make([]int, len(rawCitiesDiff))
		generals := make([]int, len(rawGenerals))
		for i, v := range rawMapDiff {
			mapDiff[i] = int(v.(float64))
		}
		for i, v := range rawCitiesDiff {
			citiesDiff[i] = int(v.(float64))
		}
		for i, v := range rawGenerals {
			generals[i] = int(v.(float64))
		}
		bt.state.update(mapDiff, citiesDiff, generals)
		from, to, half := bt.state.move()
		bt.cl.Emit("attack", from, to, half)

		bt.turn += 1
	})

	bt.cl.On("game_over", func(data ...interface{}) {
		log.Println("game ended")
		bt.cl.Disconnect()
	})
}

func (bt *Bot) JoinCustom(room string, private bool) error {
	bt.cl.Emit("join_private", room, bt.id, bt.token)
	bt.cl.Emit("chat_message", "chat_custom_queue_"+room, bt.initmsg)
	go func() {
		time.Sleep(1 * time.Second)
		if !private {
			bt.cl.Emit("make_custom_public", room)
		}
		bt.cl.Emit("set_custom_options", room, map[string]interface{}{"game_speed": 2})
	}()
	return nil
}

func (bt *Bot) JoinFFA() error {
	if bt.server != BOT {
		return fmt.Errorf("FFA only available in bot server")
	}
	bt.cl.Emit("play", bt.id)
	return nil
}

func (bt *Bot) Join1v1() error {
	if bt.server != BOT {
		return fmt.Errorf("1v1 only available in bot server")
	}
	bt.cl.Emit("join_1v1", bt.id)
	return nil
}

func (bt *Bot) Listen() error {
	return bt.cl.Listen()
}
