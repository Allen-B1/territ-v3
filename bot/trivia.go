package bot

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
)

type Trivia struct {
	scores   map[string]int
	question string
	qScore   int
	answer   string
	guessed  map[string]bool
}

type tdbResults struct {
	Results []tdbItem `json:"results"`
}

type tdbItem struct {
	Question   string `json:"question"`
	Answer     string `json:"correct_answer"`
	Difficulty string `json:"difficulty"`
}

func NewTrivia() *Trivia {
	t := new(Trivia)
	t.scores = make(map[string]int)
	t.guessed = make(map[string]bool)
	err := t.next()
	if err != nil {
		log.Println(err)
	}
	return t
}

func (t *Trivia) Skip() (string, error) {
	if len(t.guessed) < 2 {
		return "", fmt.Errorf("At least 2 people must guess to skip")
	}

	answer := t.answer
	t.next()
	return answer, nil
}

func (t *Trivia) next() error {
	resp, err := http.Get("https://opentdb.com/api.php?amount=1")
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var out tdbResults
	err = json.Unmarshal(body, &out)
	if err != nil {
		return err
	}

	t.question = html.UnescapeString(out.Results[0].Question)
	t.answer = html.UnescapeString(out.Results[0].Answer)
	if out.Results[0].Difficulty == "hard" {
		t.qScore = 3
	} else if out.Results[0].Difficulty == "medium" {
		t.qScore = 2
	} else {
		t.qScore = 1
	}

	return nil
}

func (t *Trivia) Scores() (map[string]int, []string) {
	out := make([]string, 0)
	for player, _ := range t.scores {
		if len(player) != 0 {
			out = append(out, player)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return t.scores[out[i]] > t.scores[out[j]]
	})
	return t.scores, out
}

func (t *Trivia) Question() (string, int) {
	return t.question, t.qScore
}

func (t *Trivia) Guess(answer string, person string) string {
	if strings.EqualFold(answer, t.answer) {
		t.scores[person] += t.qScore
		answer := t.answer
		t.next()
		t.guessed = make(map[string]bool)
		return answer
	} else {
		t.guessed[person] = true
	}
	return ""
}
