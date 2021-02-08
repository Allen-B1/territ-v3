package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/allenb123/socketio"
	"github.com/allen-b1/territ-v3/bot/alg"
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

func isBot(username string) bool {
	return strings.HasPrefix(username, "territ") || strings.HasPrefix(username, "[Bot]") || strings.HasPrefix(username, "[BOT]") || strings.Contains(username, "myssix")
}

type Bot struct {
	cl *socketio.Client

	server Server
	id     string
	token  string

	initmsg string

	username string
	number   int
	isHost   bool

	settings settings

	// map chat => trivia :)
	trivia map[string]*Trivia

	surrenderRequests map[string]bool

	customNeedSettings bool
	customPrivate      bool
	customRoom         string
	customNumForce     int

	// Information about the room
	// No way to tell *which* room, unfortunately :(
	room roomInfo

	// Information about the game
	alg     alg.Alg
	algType string
	turn    int
}

func New(server Server, id string, token string) (*Bot, error) {
	bt := new(Bot)
	bt.id = id
	bt.token = token
	bt.server = server
	bt.initmsg = "Hi! #" + fmt.Sprint(time.Now().UnixNano()%10000)
	bt.number = 1

	bt.surrenderRequests = make(map[string]bool)

	bt.room = roomInfo{}
	bt.room.teams = make(map[string]int)

	bt.settings.Init()

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

		humanCount := 0
		botCount := 0
		for player, team := range bt.room.teams {
			if team > 12 {
				continue
			}
			if isBot(player) {
				botCount += 1
			} else {
				humanCount += 1
			}
		}

		switch strings.ToLower(fields[0]) {
		case "sh":
			if len(fields) > 1 {
				switch fields[1] {
				case "echo":
					bt.cl.Emit("chat_message", chatroom, strings.Join(fields[2:], " "))
				case "whoami":
					go func(username string) {
						bt.cl.Emit("chat_message", chatroom, "username: "+username)
						time.Sleep(500 * time.Millisecond)
						bt.cl.Emit("chat_message", chatroom, "bot: "+fmt.Sprint(isBot(username)))
					}(m["username"].(string))
				case "pwd":
					if room != "" {
						bt.cl.Emit("chat_message", chatroom, "/games/"+room)
					} else {
						bt.cl.Emit("chat_message", chatroom, "pwd: insufficient permissions")
					}
				case "cd":
					dir := strings.Join(fields[2:], " ")
					if dir == "/" && room != "" {
						bt.cl.Disconnect()
					} else if dir == "/" || dir == "/games" || dir == "/games/" {
						bt.cl.Emit("chat_message", chatroom, "cd: insufficient permissions")
					} else {
						bt.cl.Emit("chat_message", chatroom, "cd: no such file or directory")
					}
				default:
					bt.cl.Emit("chat_message", chatroom, fields[1]+": unknown command")
				}
			}

		case "force":
			required := humanCount/2 + 1
			if humanCount <= 1 {
				required = 0
			}
			if bt.customNumForce >= required || author == "Lazerpent" || author == "person2597" {
				bt.cl.Emit("set_force_start", room, true)
			} else {
				bt.cl.Emit("chat_message", chatroom, fmt.Sprintf("not enough force (%d / %d)", bt.customNumForce, required))
			}

		case "alg":
			if bt.alg == nil {
				if strings.EqualFold(strings.Join(fields[1:], " "), "path") {
					bt.algType = "path"
					bt.cl.Emit("chat_message", chatroom, "Set algorithm to Path")
				} else {
					bt.algType = "random"
					bt.cl.Emit("chat_message", chatroom, "Set algorithm to Random")
				}
			} else if len(fields) > 1 {
				msg := bt.alg.Command(fields[1])
				bt.cl.Emit("chat_message", chatroom, msg)
			}

		// settings
		case "speed":
			if !bt.isHost || isBot(m["username"].(string)) {
				break
			}
			speed := 4
			if len(fields) >= 2 {
				speed, _ = strconv.Atoi(fields[1])
			}
			bt.cl.Emit("set_custom_options", room, map[string]interface{}{"game_speed": speed})

		case "players":
			if !bt.isHost || isBot(m["username"].(string)) {
				break
			}
			speed := 4
			if len(fields) >= 2 {
				speed, _ = strconv.Atoi(fields[1])
			}
			bt.cl.Emit("set_custom_options", room, map[string]interface{}{"max_players": speed})

		case "map":
			if !bt.isHost || isBot(m["username"].(string)) {
				break
			}
			mapName := strings.Join(fields[1:], " ")
			newMapName, err := bt.settings.Vote(fmt.Sprint(m["username"]), mapName)
			if err == settingsInvalidMap {
				bt.cl.Emit("chat_message", chatroom, mapName+"' is not a map")
				break
			} else if err == settingsInvalidCommand {
				bt.cl.Emit("chat_message", chatroom, "valid special maps are :random, :empty")
				break
			} else if mapName != newMapName && mapName != "" {
				bt.cl.Emit("chat_message", chatroom, fmt.Sprint(m["username"])+"'s vote set to '"+newMapName+"' (since '"+mapName+"' is not a map)")
			} else {
				bt.cl.Emit("chat_message", chatroom, fmt.Sprint(m["username"])+"'s vote set to '"+newMapName+"'")
			}

			bt.cl.Emit("set_custom_options", room, bt.settings.Settings(nil))
		case "votes":
			if !bt.isHost || isBot(m["username"].(string)) {
				break
			}
			count := bt.settings.Count()
			go func() {
				for map_, votes := range count {
					time.Sleep(500 * time.Millisecond)
					if map_ == "" {
						map_ = ":empty"
					}
					bt.cl.Emit("chat_message", chatroom, map_+" - "+fmt.Sprint(votes))
				}
			}()
		case "empty":
			if !bt.isHost || isBot(m["username"].(string)) {
				break
			}
			bt.settings.Vote(fmt.Sprint(m["username"]), "")
			bt.cl.Emit("set_custom_options", room, bt.settings.Settings(nil))
		case "mountain", "mountains":
			if !bt.isHost || isBot(m["username"].(string)) {
				break
			}
			if len(fields) > 1 {
				n, _ := strconv.Atoi(fields[1])
				bt.settings.VoteMountain(fmt.Sprint(m["username"]), n)
				bt.cl.Emit("set_custom_options", room, bt.settings.Settings([]string{"mountain_density"}))
			}
		case "swamp", "swamps":
			if !bt.isHost || isBot(m["username"].(string)) {
				break
			}
			if len(fields) > 1 {
				n, _ := strconv.Atoi(fields[1])
				bt.settings.VoteSwamp(fmt.Sprint(m["username"]), n != 0)
				bt.cl.Emit("set_custom_options", room, bt.settings.Settings([]string{"swamp_density"}))
			}
		case "city", "cities":
			if !bt.isHost || isBot(m["username"].(string)) {
				break
			}
			if len(fields) > 1 {
				n, _ := strconv.Atoi(fields[1])
				bt.settings.VoteCity(fmt.Sprint(m["username"]), n != 0)
				bt.cl.Emit("set_custom_options", room, bt.settings.Settings([]string{"city_density"}))
			}
		case "size":
			if !bt.isHost || isBot(m["username"].(string)) {
				break
			}
			if len(fields) > 1 {
				n, _ := strconv.ParseFloat(fields[1], 64)
				bt.settings.SetSize(n)
				bt.cl.Emit("set_custom_options", room, bt.settings.Settings([]string{"width", "height"}))
			}

		case "team":
			bt.cl.Emit("set_custom_team", room, bt.room.teams[author])
		case "help":
			go func() {
				msgs := []string{
					"Hi! I'm a terrible bot. Possible commands:",
					"* /trivia",
					"* /team",
					"Source code: https://github.com/allen-b1/territ-v3",
				}
				if bt.isHost && room != "" {
					msgs = []string{
						"Hi! I'm a terrible bot. Possible commands:",
						"* /force",
						"* /speed [1|2|3|4]",
						"* /map MAP",
						"* /team",
						"Source code: https://github.com/allen-b1/territ-v3",
					}
				}

				for _, msg := range msgs {
					bt.cl.Emit("chat_message", chatroom, msg)
					time.Sleep(500 * time.Millisecond)
				}
			}()

		case "public":
			if bt.isHost {
				bt.cl.Emit("make_custom_public", room)
			}

		case "surrender":
			if room == "" {
				if isBot(m["username"].(string)) {
					break
				}

				bt.surrenderRequests[m["username"].(string)] = true
				amountRequired := humanCount/2 + 1
				amountRequests := len(bt.surrenderRequests)

				bt.cl.Emit("chat_message", chatroom, fmt.Sprintf(m["username"].(string)+" requested surrender (%d / %d)", amountRequests, amountRequired))
				if amountRequests >= amountRequired {
					bt.cl.Emit("surrender")
				}
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
			switch strings.ToLower(fields[1]) {
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

		players, ok := m["usernames"].([]interface{})
		if ok {
			teams := m["teams"].([]interface{})
			for i := 0; i < len(players); i++ {
				if players[i] != nil {
					bt.room.teams[fmt.Sprint(players[i])] = int(teams[i].(float64))
				}
			}
		}

		bt.isHost = false
		if len(players) > 0 {
			if name, ok := players[0].(string); ok && name == bt.username {
				bt.isHost = true
			}
		}

		forceNum, ok := m["numForce"].([]interface{})
		if ok {
			bt.customNumForce = len(forceNum)
		}

		if bt.customNeedSettings {
			bt.cl.Emit("chat_message", "chat_custom_queue_"+bt.customRoom, bt.initmsg)
			if !bt.customPrivate {
				bt.cl.Emit("make_custom_public", bt.customRoom)
			}
			bt.cl.Emit("update_custom_chat_recording", bt.customRoom, nil, false)
			bt.cl.Emit("set_custom_options", bt.customRoom, map[string]interface{}{"speed": 4})
			bt.cl.Emit("set_custom_options", bt.customRoom, bt.settings.Settings(nil))
			bt.customNeedSettings = false
		}
	})

	bt.cl.On("ping_tile", func(data ...interface{}) {
		tile := data[0].(float64)
		bt.alg.Ping(int(tile))
	})

	bt.cl.On("error_set_username", func(data ...interface{}) {
		log.Println(data[0])
	})

	bt.cl.On("game_start", func(data ...interface{}) {
		m := data[0].(map[string]interface{})
		log.Println("game started with", m["usernames"])

		playerIndex := int(m["playerIndex"].(float64))
		teamsMap := make(map[int]bool)
		teams, ok := m["teams"].([]interface{})
		if ok {
			for player, team := range teams {
				if team == teams[playerIndex] {
					teamsMap[player] = true
				}
			}
		}
		teamsMap[playerIndex] = true

		rawSwamps := m["swamps"].([]interface{})
		swamps := make([]int, len(rawSwamps))
		for i, swamp := range rawSwamps {
			swamps[i] = int(swamp.(float64))
		}

		map_ := alg.NewMap(swamps)
		if bt.algType == "path" {
			bt.alg = new(alg.Path).Init(map_, playerIndex, teamsMap)
		} else {
			bt.alg = new(alg.Random).Init(map_, playerIndex, teamsMap)
		}
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
			vf, ok := v.(float64)
			if !ok {
				vf = -1
			}
			mapDiff[i] = int(vf)
		}
		for i, v := range rawCitiesDiff {
			vf, ok := v.(float64)
			if !ok {
				vf = -1
			}
			citiesDiff[i] = int(vf)
		}
		for i, v := range rawGenerals {
			vf, ok := v.(float64)
			if !ok {
				vf = -1
			}
			generals[i] = int(vf)
		}
		bt.alg.Map().Update(mapDiff, citiesDiff, generals)
		from, to, half := bt.alg.Move()
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
	bt.customNeedSettings = true
	bt.customPrivate = private
	bt.customRoom = room
	return nil
}

func (bt *Bot) Join2v2(team string) error {
	if bt.server != BOT {
		return fmt.Errorf("2v2 only available in bot server")
	}
	bt.cl.Emit("join_team", team, bt.id, bt.token)
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

func (bt *Bot) SetUsername(username string) error {
	return bt.cl.Emit("set_username", bt.id, username)
}
