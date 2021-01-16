package alg

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
}
