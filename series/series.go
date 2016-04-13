package series

import "sync"

type Series struct {
	size         int
	data         []float64
	level        int
	first        float64
	last         float64
	current      float64
	currentLevel int

	mutex sync.Mutex
}

func NewSeries(size int) *Series {
	res := &Series{size: size, data: make([]float64, 0), level: 1, first: 0, last: 0, current: 0, currentLevel: 0}
	return res
}

func (s *Series) Size() int {
	if len(s.data) < s.size {
		return len(s.data)
	} else {
		return s.size
	}
}

func (s *Series) Push(value float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.current += value
	s.currentLevel++
	s.last = value
	if len(s.data) == 0 {
		s.first = value
	}

	if s.currentLevel == s.level {
		s.data = append(s.data, s.current/float64(s.level))
		s.current = 0
		s.currentLevel = 0
	}

	if len(s.data) == s.size*2 {
		data := make([]float64, s.size)
		for i := 0; i < s.size; i++ {
			data[i] = (s.data[i*2] + s.data[i*2+1]) / 2
		}
		s.data = data
		s.level *= 2
	}
}

func (s *Series) Get() []float64 {
	var res []float64
	if len(s.data) < s.size {
		res = make([]float64, len(s.data))
	} else {
		res = make([]float64, s.size)
	}

	for i, p := 0, 0; i < len(s.data); p++ {
		sum := 0.
		count := 0.
		for ; float64(i) <= float64(len(s.data))/float64(len(res))*float64(p) && i < len(s.data); i++ {
			count++
			sum += s.data[i]
		}
		if p < len(res) {
			res[p] = sum / count
		}
	}
	if len(res) > 0 {
		res[0] = s.first
	}
	if len(res) > 1 {
		res[len(res)-1] = s.last
	}
	return res
}
