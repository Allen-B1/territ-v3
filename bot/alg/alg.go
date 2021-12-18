package alg

const (
	TerrainEmpty    = -1
	TerrainMountain = -2
	TerrainFog      = -3
	TerrainObstacle = -4
)

type Map struct {
	swamps   map[int]bool
	map_     []int
	cities   []int
	generals []int
}

func NewMap(swamps []int) *Map {
	m := new(Map)
	m.swamps = make(map[int]bool)
	for _, swamp := range swamps {
		m.swamps[swamp] = true
	}
	return m
}

func (m *Map) Update(mapDiff []int, citiesDiff []int, generals []int) {
	m.map_ = patch(m.map_, mapDiff)
	m.cities = patch(m.cities, citiesDiff)
	m.generals = generals
}

func (m *Map) terrain() []int {
	width := m.map_[0]
	height := m.map_[1]
	size := width * height
	return m.map_[size+2 : size*2+2]
}

func (m *Map) armies() []int {
	width := m.map_[0]
	height := m.map_[1]
	size := width * height
	return m.map_[2 : size+2]
}

func (m *Map) dimen() (int, int) {
	return m.map_[0], m.map_[1]
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

type Alg interface {
	Init(map_ *Map, playerIndex int, allies map[int]bool) Alg
	Map() *Map
	Move() (int, int, bool)
	Ping(tile int)
	Command(cmd string) string
}
