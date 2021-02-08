package alg

import (
	"math/rand"
)

func weightedRandom(weights []int) int {
	i := 0
	boundaries := []int{0}
	for _, weight := range weights {
		i += weight
		boundaries = append(boundaries, i)
	}
	n := rand.Intn(i)
	for i := 0; i < len(boundaries)-1; i++ {
		if n >= boundaries[i] && n < boundaries[i+1] {
			return i
		}
	}
	return -1
}

type Random struct {
	map_        *Map
	playerIndex int
	allies      map[int]bool
	pinged      int

	turn  int
	order []int

	disableRandom bool
}

func (a *Random) Init(map_ *Map, playerIndex int, allies map[int]bool) Alg {
	a.map_ = map_
	a.playerIndex = playerIndex
	a.allies = allies
	a.pinged = -1
	a.turn = 0
	a.order = []int{0, 1, 2, 3}
	return a
}
func (a *Random) Map() *Map {
	return a.map_
}
func (a *Random) Move() (int, int, bool) {
	a.turn += 1
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

	// Set direction to move in
	if a.turn%25 == 0 {
		order := rand.Intn(4)
		a.order[0] = order
		a.order[1] = (order + 1) % 4
		a.order[2] = (order + 3) % 4
		a.order[3] = (order + 2) % 4
	}

	// Conquer generals
	for i := 0; i < size; i++ {
		if terrain[i] == a.playerIndex && armies[i] >= 2 {
			possibles := []int{}
			adjs := adjacentTiles(i, width, height)
			for _, adj := range adjs {
				if adj != -1 && !a.allies[terrain[adj]] && terrain[adj] != -2 && armies[i] > armies[adj]+1 && !swamps[adj] && isGeneral[adj] {
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
			if adj != -1 && terrain[adj] == a.playerIndex && (a.allies[terrain[pinged]] || armies[adj] > armies[pinged]+1) {
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
				if adj != -1 && !a.allies[terrain[adj]] && terrain[adj] != a.playerIndex && terrain[adj] != -2 && armies[i] > armies[adj]+1 && !swamps[adj] && cities[adj] {
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
			adjs := adjacentTiles(i, width, height)
			for _, ind := range a.order {
				adj := adjs[ind]
				if adj != -1 && !a.allies[terrain[adj]] && terrain[adj] != -2 && armies[i] > armies[adj]+1 && !swamps[adj] {
					return i, adj, false
				}
			}
		}
	}

	// Move randomly
	if a.disableRandom {
		return 0, 0, false
	}

	half := false
	possibleFrom := make([]int, 0)
	for i := 0; i < size; i++ {
		if terrain[i] == a.playerIndex && armies[i] >= 2 {
			adjs := adjacentTiles(i, width, height)
			for _, adj := range adjs {
				if adj != -1 && a.allies[terrain[adj]] && (terrain[adj] == a.playerIndex || armies[adj] < 10) {
					possibleFrom = append(possibleFrom, i)
					break
				}
			}
		}
	}
	if len(possibleFrom) > 0 {
		possibleArmies := make([]int, len(possibleFrom))
		for i, tile := range possibleFrom {
			possibleArmies[i] = armies[tile]
		}
		tile := possibleFrom[weightedRandom(possibleArmies)]
		adjs := adjacentTiles(tile, width, height)

		toTile := 0
		for _, ind := range a.order {
			adj := adjs[ind]
			if adj != -1 && a.allies[terrain[adj]] && (terrain[adj] == a.playerIndex || armies[adj] < 10) {
				toTile = adj
				break
			}
		}

		if generals[a.playerIndex] == tile && armies[tile] >= 1000 {
			half = true
		}

		return tile, toTile, half
	}

	return 0, 0, false
}

func (a *Random) Command(cmd string) string {
	if cmd == "stop" {
		a.disableRandom = true
		return "Disabled random movement"
	} else if cmd == "start" {
		a.disableRandom = false
		return "Enabled random movement"
	}
	return ""
}

func (a *Random) Ping(tile int) {
	a.pinged = tile
}

func adjacentTiles(tile int, width int, height int) []int {
	row := (tile / width) | 0
	col := tile % width
	out := []int{-1, -1, -1, -1}

	if col < width-1 {
		out[0] = tile + 1
	}
	if col > 0 {
		out[1] = tile - 1
	}
	if row < height-1 {
		out[2] = tile + width
	}
	if row > 0 {
		out[3] = tile - width
	}

	return out
}
