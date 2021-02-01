package alg

type Path struct {
	map_        *Map
	playerIndex int
	allies      map[int]bool

	turn int

	expandDirection int
	collectPath     []int

	//states:
	// start int
	// expand_line int
	// move_border
}

func (a *Path) Init(map_ *Map, playerIndex int, allies map[int]bool) Alg {
	a.map_ = map_
	a.playerIndex = playerIndex
	a.allies = allies

	return a
}
func (a *Path) Map() *Map {
	return a.map_
}
func (a *Path) Command(string) string { return "" }

func (a *Path) collect() (int, int, bool) {
	return 0, 0, false
}

func (a *Path) expand() (int, int, bool) {
	if a.turn < 12*2+1 {
		return 0, 0, false
	}

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
	_ = swamps

	direction := a.expandDirection

	fromTile := -1
	for tile, tterrain := range terrain {
		if tterrain == a.playerIndex && armies[tile] > 1 && generals[a.playerIndex] != tile {
			adjs := adjacentTiles(tile, width, height)
			for _, adjTile := range adjs {
				if adjTile != -1 && armies[adjTile] == 0 && terrain[adjTile] != -2 {
					fromTile = tile
					break
				}
			}
		}
	}

	if fromTile == -1 {
		fromTile = generals[a.playerIndex]

		adjs := adjacentTiles(fromTile, width, height)
		hasAdjTile := false
		for _, adjTile := range adjs {
			if adjTile != -1 && armies[adjTile] == 0 {
				hasAdjTile = true
				break
			}
		}

		if hasAdjTile {
			direction = (direction + 1) % 4
			a.expandDirection = direction
		} else {
			// done expanding
		}
	}
	adjs := adjacentTiles(fromTile, width, height)
	for i := 0; i < 4; i++ {
		adjTile := adjs[(direction+i)%4]
		if adjs[(direction+i)%4] != -1 && armies[adjTile] == 0 && terrain[adjTile] != -2 {
			return fromTile, adjs[(direction+i)%4], false
		}
	}

	return 0, 0, false
}

func (a *Path) Move() (int, int, bool) {
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
	_ = swamps
	_ = armies
	_ = terrain
	_ = generals

	if a.turn < 50 {
		return a.expand()
	}
	if a.turn > 50 {

	}

	return 0, 0, false
}
func (a *Path) Ping(tile int) {}
