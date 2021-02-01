package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
)

var (
	settingsInvalidMap     = errors.New("invalid map for devs only")
	settingsInvalidCommand = errors.New("invalid command for devs only")
)

type settings struct {
	mapVotes        map[string]string
	cityDensity     map[string]bool
	mountainDensity map[string]int
	swampDensity    map[string]bool
	size            float64
}

func (m *settings) randomMap() string {
	list := "top"
	resp, err := http.Get("http://generals.io/api/maps/lists/" + list)
	if err != nil {
		log.Println(err)
		return ""
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return ""
	}
	out := make([]map[string]interface{}, 0)
	json.Unmarshal(body, &out)
	if len(out) != 0 {
		map_ := fmt.Sprint(out[rand.Intn(len(out))]["title"])
		return map_
	}
	return ""
}

func (m *settings) SetSize(size float64) {
	m.size = size
}

func (m *settings) getMap(map_ string) string {
	resp, err := http.Get("http://generals.io/api/maps/search?q=" + url.QueryEscape(map_))
	if err != nil {
		log.Println(err)
		return ""
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return ""
	}
	out := make([]map[string]interface{}, 0)
	json.Unmarshal(body, &out)
	if len(out) == 0 {
		return ""
	}
	for _, mapobj := range out {
		if mapobj["title"] == map_ {
			return map_
		}
	}
	return out[0]["title"].(string)
}

func (m *settings) Init() {
	m.mapVotes = make(map[string]string)
	m.cityDensity = make(map[string]bool)
	m.swampDensity = make(map[string]bool)
	m.mountainDensity = make(map[string]int)
}

func (m *settings) Vote(player string, map_ string) (string, error) {
	if map_ == "" {
		m.mapVotes[player] = ""
		return ":empty", nil
	} else if map_[0] == ':' {
		if map_ != ":empty" && map_ != ":random" {
			return "", settingsInvalidCommand
		}

		if map_ == ":empty" {
			m.mapVotes[player] = ""
		} else {
			m.mapVotes[player] = map_
		}
		return map_, nil
	} else {
		mapName := m.getMap(map_)
		if mapName == "" {
			return "", settingsInvalidMap
		}
		m.mapVotes[player] = mapName
		return mapName, nil
	}
}

func (m *settings) VoteCity(player string, cities bool) {
	m.cityDensity[player] = cities
}

func (m *settings) VoteMountain(player string, mountain int) {
	if mountain > 3 {
		mountain = 3
	}
	if mountain < 0 {
		mountain = 1
	}
	m.mountainDensity[player] = mountain
}

func (m *settings) VoteSwamp(player string, swamp bool) {
	m.swampDensity[player] = swamp
}

func (m *settings) Count() map[string]int {
	out := make(map[string]int)
	for _, map_ := range m.mapVotes {
		out[map_] += 1
	}
	return out
}

func (m *settings) winnerCity() bool {
	out := make(map[bool]int)
	for _, setting := range m.cityDensity {
		out[setting] += 1
	}

	if out[false] > out[true] {
		return false
	}
	return true
}
func (m *settings) winnerSwamp() bool {
	out := make(map[bool]int)
	for _, setting := range m.swampDensity {
		out[setting] += 1
	}

	if out[true] > out[false] {
		return true
	}
	return false
}
func (m *settings) winnerMountain() int {
	out := make(map[int]int)
	for _, setting := range m.mountainDensity {
		out[setting] += 1
	}

	if out[0] > out[1] && out[0] > out[2] {
		return 0
	}
	if out[2] > out[1] && out[2] > out[0] {
		return 2
	}
	return 1
}

func (m *settings) Settings(props []string) map[string]interface{} {
	count := m.Count()
	winnerVotes := 0
	winner := ":random"
	for map_, votes := range count {
		if votes > winnerVotes {
			winnerVotes = votes
			winner = map_
		}
	}

	original := (map[string]interface{})(nil)

	if winner == ":random" {
		original = map[string]interface{}{"map": m.randomMap()}
	} else if winner != "" {
		original = map[string]interface{}{"map": winner}
	} else {
		b2i := map[bool]int{false: 0, true: 1}

		city := b2i[m.winnerCity()]
		swamp := b2i[m.winnerSwamp()]
		mountain := map[int]float64{0: 0.0, 1: 0.2, 2: 0.5, 3: 1.0}[m.winnerMountain()]
		original = map[string]interface{}{
			"map":              "",
			"city_density":     city,
			"swamp_density":    swamp,
			"mountain_density": mountain,
			"width":            m.size,
			"height":           m.size,
		}
	}

	if props == nil {
		return original
	}

	out := make(map[string]interface{})
	for _, prop := range props {
		if value, ok := original[prop]; ok {
			out[prop] = value
		}
	}
	return out
}
