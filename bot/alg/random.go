package alg

import (
	"math/rand"
)

type Random struct {
	map_ *Map
	playerIndex int
	allies      map[int]bool
	pinged      int
}

func (a *Random) Init(map_ *Map, playerIndex int, allies map[int]bool) Alg {
	a.map_ = map_
	a.playerIndex = playerIndex
	a.allies = allies
	a.pinged = -1
	return a
}
func (a *Random) Map() *Map {
	return a.map_
}
func (a *Random) Move() (int, int, bool) {
	width := a.map_.map_[0]
	height := a.map_.map_[1]
	size := width * height
	armies := a.map_.map_[2 : size+2]
	terrain := a.map_.map_[size+2 : size*2+2]
	cities := make(map[int]bool)
	for _, city := range a.map_.cities {
		cities[city] = true
	}
	generals := a.map_.generals
	swamps := a.map_.swamps

	isGeneral := make(map[int]bool)
	for _, general := range generals {
		isGeneral[general] = true
	}

	// Conquer generals
	for i := 0; i < size; i++ {
		if terrain[i] == a.playerIndex && armies[i] >= 2 {
			possibles := []int{}
			adjs := adjacentTiles(i, width, height)
			for _, adj := range adjs {
				if !a.allies[terrain[adj]] && terrain[adj] != -2 && armies[i] > armies[adj]+1 && !swamps[adj] && isGeneral[adj] {
					possibles = append(possibles, adj)
				}
			}

			if len(possibles) > 0 {
				return i, possibles[rand.Intn(len(possibles))], false
			}
		}
	}

	pinged := a.pinged
	if pinged != -1 {
		adjs := adjacentTiles(pinged, width, height)
		maxValue := 0
		maxTile := -1
		for _, adj := range adjs {
			if terrain[adj] == a.playerIndex && (a.allies[terrain[pinged]] || armies[adj] > armies[pinged]+1) {
				if armies[adj] > maxValue {
					maxValue = armies[adj]
					maxTile = adj
				}
			}
		}

		if maxTile != -1 {
			a.pinged = -1
			return maxTile, pinged, false
		}
	}

	// Conquer cities
	for i := 0; i < size; i++ {
		if terrain[i] == a.playerIndex && armies[i] >= 2 {
			possibles := []int{}
			adjs := adjacentTiles(i, width, height)
			for _, adj := range adjs {
				if !a.allies[terrain[adj]] && terrain[adj] != -2 && armies[i] > armies[adj]+1 && !swamps[adj] && cities[adj] {
					possibles = append(possibles, adj)
				}
			}

			if len(possibles) > 0 {
				return i, possibles[rand.Intn(len(possibles))], false
			}
		}
	}

	// Conquer tiles
	for i := 0; i < size; i++ {
		if terrain[i] == a.playerIndex && armies[i] >= 2 {
			possibles := []int{}
			adjs := adjacentTiles(i, width, height)
			for _, adj := range adjs {
				if !a.allies[terrain[adj]] && terrain[adj] != -2 && armies[i] > armies[adj]+1 && !swamps[adj] {
					possibles = append(possibles, adj)
				}
			}

			if len(possibles) > 0 {
				return i, possibles[rand.Intn(len(possibles))], false
			}
		}
	}

	// Move randomly
	for i := 0; i < 256; i++ {
		i := rand.Intn(size)
		if terrain[i] == a.playerIndex && armies[i] >= 2 {
			if generals[a.playerIndex] == i && armies[i] >= 30 {
				continue
			}

			adjs := adjacentTiles(i, width, height)
			rand.Shuffle(len(adjs), func(i, j int) {
				adjs[i], adjs[j] = adjs[j], adjs[i]
			})
			for _, adj := range adjs {
				if a.allies[terrain[adj]] {
					return i, adj, false
				}
			}
		}
	}

	return 0, 0, false
}

func (a *Random) Ping(tile int) {
	a.pinged = tile
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
