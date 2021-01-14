package bot

import (
	"math/rand"
)

type state struct {
	swamps      map[int]bool
	map_        []int
	cities      []int
	generals    []int
	playerIndex int
	allies      map[int]bool
	pinged    int
}

func newState(playerIndex int, allies map[int]bool) *state {
	s := new(state)
	s.playerIndex = playerIndex
	s.allies = allies
	s.pinged = -1
	return s
}

func patch(old []int, diff []int) []int {
	out := make([]int, 0)
	for i := 0; i < len(diff); {
		if diff[i] != 0 {
			out = append(out, old[len(out):len(out)+diff[i]]...)
		}
		i++
		if i < len(diff) && diff[i] != 0 {
			out = append(out, diff[i+1:i+1+diff[i]]...)
			i += diff[i]
		}
		i++
	}
	return out
}

func (s *state) init(swamps map[int]bool) {
	s.swamps = swamps
}

func (s *state) update(mapDiff []int, citiesDiff []int, generals []int) {
	s.map_ = patch(s.map_, mapDiff)
	s.cities = patch(s.cities, citiesDiff)
	s.generals = generals
}
func (s *state) ping(tile int) {
	s.pinged = tile
}

func (s *state) move() (int, int, bool) {
	width := s.map_[0]
	height := s.map_[1]
	size := width * height
	armies := s.map_[2 : size+2]
	terrain := s.map_[size+2 : size*2+2]

	pinged := s.pinged
	if pinged != -1 {
		adjs := adjacentTiles(pinged, width, height)
		maxValue := 0
		maxTile := -1
		for _, adj := range adjs {
			if terrain[adj] == s.playerIndex && (s.allies[terrain[pinged]] || armies[adj] > armies[pinged] + 1) {
				if armies[adj] > maxValue {
					maxValue = armies[adj]
					maxTile = adj
				}
			}
		}

		if maxTile != -1 {
			s.pinged = -1
			return maxTile, pinged, false
		}
	}

	for i := 0; i < size; i++ {
		if terrain[i] == s.playerIndex && armies[i] >= 2 {
			possibles := []int{}
			adjs := adjacentTiles(i, width, height)
			for _, adj := range adjs {
				if !s.allies[terrain[adj]] && terrain[adj] != -2 && armies[i] > armies[adj]+1 && !s.swamps[adj] {
					possibles = append(possibles, adj)
				}
			}

			if len(possibles) > 0 {
				return i, possibles[rand.Intn(len(possibles))], false
			}
		}
	}

	for i := 0; i < 256; i++ {
		i := rand.Intn(size)
		if terrain[i] == s.playerIndex && armies[i] >= 2 {
			if s.generals[s.playerIndex] == i && armies[i] >= 30 {
				continue
			}

			adjs := adjacentTiles(i, width, height)
			rand.Shuffle(len(adjs), func(i, j int) {
				adjs[i], adjs[j] = adjs[j], adjs[i]
			})
			for _, adj := range adjs {
				if s.allies[terrain[adj]] {
					return i, adj, false
				}
			}
		}
	}

	return 0, 0, false
}

func adjacentTiles(tile int, width int, height int) []int {
	row := (tile / width) | 0
	col := tile % width
	out := make([]int, 0)

	if col < width-1 {
		out = append(out, tile+1)
	}
	if col > 0 {
		out = append(out, tile-1)
	}
	if row < height-1 {
		out = append(out, tile+width)
	}
	if row > 0 {
		out = append(out, tile-width)
	}

	return out
}
