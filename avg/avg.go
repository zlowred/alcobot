package avg

type Avg struct {
	unsorted   []int16
	sorted     []int16
	ptr        int16
	mPtr       int16
	sampleSize int16
	windowSize int16
	Counter    int
	Ready      bool
}

func NewAvg(windowSize int, sampleSize int) *Avg {
	avg := &Avg{unsorted: make([]int16, windowSize), sorted: make([]int16, windowSize), ptr: int16(windowSize), mPtr: int16(windowSize / 2), sampleSize: int16(sampleSize), windowSize: int16(windowSize), Ready: false}
	return avg
}

func (a *Avg) ResetCounter() {
	a.Counter = 0
	a.Ready = false
}

func (a *Avg) Add(val int16) {
	a.Counter++
	if a.Counter >= len(a.unsorted) {
		a.Ready = true
	}

	if a.ptr == 0 {
		a.ptr = a.windowSize
	}

	a.ptr--

	old := a.unsorted[a.ptr]

	if old == val {
		return
	}

	a.unsorted[a.ptr] = val

	i := a.windowSize

	for i > 0 {
		i--
		if old == a.sorted[i] {
			break
		}
	}

	a.sorted[i] = val

	if val > old {
		for p, q := i, i+1; q < a.windowSize; p, q = p+1, q+1 {
			if a.sorted[p] > a.sorted[q] {
				tmp := a.sorted[p]
				a.sorted[p] = a.sorted[q]
				a.sorted[q] = tmp
			} else {
				return
			}
		}
	} else {
		for p, q := i-1, i; q > 0; p, q = p-1, q-1 {
			if a.sorted[p] > a.sorted[q] {
				tmp := a.sorted[p]
				a.sorted[p] = a.sorted[q]
				a.sorted[q] = tmp
			} else {
				return
			}
		}
	}
}

func (a *Avg) Median() int16 {
	return a.sorted[a.mPtr]
}

func (a *Avg) Average() int16 {
	left := a.mPtr - a.sampleSize/2
	right := left + a.sampleSize

	var sum int32 = 0

	for _, x := range a.sorted[left:right] {
		sum += int32(x)
	}
	return int16(sum / int32(a.sampleSize))
}
