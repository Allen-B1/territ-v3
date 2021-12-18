package alg

import (
	"log"
)

type Path struct {
	map_        *Map
	playerIndex int
	allies      map[int]bool

	turn int

	// first round
	r1targetTile  int
	r1currentTile int
	r1stage       int
}

func (a *Path) Init(map_ *Map, playerIndex int, allies map[int]bool) Alg {
	a.map_ = map_
	a.playerIndex = playerIndex
	a.allies = allies

	a.r1targetTile = -1
	a.r1currentTile = -1

	return a
}
func (a *Path) Map() *Map {
	return a.map_
}
func (a *Path) Command(string) string { return "" }

func (a *Path) Move() (int, int, bool) {
	w, h := a.map_.dimen()
	general := a.map_.generals[a.playerIndex]
	terrain := a.map_.terrain()
	armies := a.map_.armies()

	a.turn += 1
	if a.turn < 50 {
		if a.r1targetTile == -1 && armies[general] >= 13 {
			generalVec := VectorFromIndex(a.map_, general)
			vec := vector{w / 2, h / 2}.Add(generalVec.Neg())
			vec = vec.Normalize((w + h) / 3)
			vec = vec.Add(generalVec)
			vec = vec.Constrain(a.map_)

			a.r1targetTile = vec.Index(a.map_)
			a.r1currentTile = general
		}

		if a.r1stage >= 3 { // after 3 tendrils, just expand
			for i := 0; i < w*h; i++ {
				if terrain[i] == a.playerIndex && armies[i] >= 2 {
					adjs := adjacentTiles(a.map_, i)
					for _, adj := range adjs {
						if !a.allies[terrain[adj]] && terrain[adj] != -2 && armies[i] > armies[adj]+1 {
							return i, adj, false
						}
					}
				}
			}
		} else if a.r1targetTile != -1 { // tendril #2 or #3 (idx 1 or 2)
			// if the tendril ends
			findNewTendril := false
			if a.r1targetTile == a.r1currentTile || armies[a.r1currentTile] <= 1 {
				findNewTendril = true
			} else { // continue on tendril
				log.Println("turn", a.turn)
				path := calculatePath(a.map_, a.r1currentTile, a.r1targetTile, func(tile int) int {
					if terrain[tile] == TerrainMountain || terrain[tile] == TerrainObstacle {
						// treat as walls
						return -1
					}
					if terrain[tile] == a.playerIndex {
						// discourage, but sometimes its impossible not to
						return 3
					}
					return 1 + armies[tile]/2
				})
				if path == nil {
					findNewTendril = true
				} else {
					a.r1currentTile = path[1]
					return path[0], path[1], false
				}
			}

			if findNewTendril {
				a.r1stage += 1

				if a.r1stage >= 3 {
					// just expand... wait... thats above
				}

				// normal vector
				generalVec := VectorFromIndex(a.map_, general)
				vec := vector{w / 2, h / 2}.Add(generalVec.Neg())
				if a.r1stage == 1 {
					vec = vec.Normal()
				} else {
					vec = vec.Normal().Neg()
				}
				vec = vec.Normalize(7)
				vec = vec.Add(generalVec)
				vec = vec.Constrain(a.map_)

				a.r1targetTile = vec.Index(a.map_)
				a.r1currentTile = general
			}
		}
	} else if a.turn < 100 {
		// collect, then attack, then expand
	}
	return 0, 0, false
}

func (a *Path) Ping(tile int) {}

type vector [2]int

func (v1 vector) Add(v2 vector) vector {
	return vector{v1[0] + v2[0], v1[1] + v2[1]}
}

func (v1 vector) Neg() vector {
	return vector{-v1[0], -v1[1]}
}

func (v1 vector) Mag() int {
	square := v1[0]*v1[0] + v1[1]*v1[1]

	est := square / 2
	for {
		if est == 0 {
			return 1
		}
		prevEst := est
		est = (est + square/est) / 2
		if est == prevEst {
			break
		}
	}

	return est
}

func (v vector) Normalize(mag int) vector {
	realMag := v.Mag()

	return vector{v[0] * mag / realMag, v[1] * mag / realMag}
}

func (v vector) Normal() vector {
	return vector{-v[1], v[0]}
}

func (v vector) Mul(n int) vector {
	return vector{v[0] * n, v[1] * n}
}

func VectorFromIndex(map_ *Map, index int) vector {
	w, _ := map_.dimen()
	return vector{index % w, index / w}
}

func (v vector) Index(map_ *Map) int {
	w, h := map_.dimen()

	if v[1]*w+v[0] > w*h {
		panic("vector out of range")
	}

	return v[1]*w + v[0]
}

func (v vector) Constrain(map_ *Map) vector {
	w, h := map_.dimen()
	if v[0] < 0 {
		v[0] = 0
	}
	if v[1] < 0 {
		v[1] = 0
	}
	if v[0] > w-1 {
		v[0] = w - 1
	}
	if v[1] > h-1 {
		v[1] = h - 1
	}
	return v
}
