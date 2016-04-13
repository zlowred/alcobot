// +build ignore
package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/zlowred/embd"
	_ "github.com/zlowred/embd/host/rpi"
	"math"
	"bufio"
	"strconv"
)

type Avg struct {
	unsorted   []int32
	sorted     []int32
	ptr        int16
	mPtr       int16
	sampleSize int16
	windowSize int16
	Counter    int
	Ready      bool
}

func NewAvg(windowSize int, sampleSize int) *Avg {
	avg := &Avg{unsorted: make([]int32, windowSize), sorted: make([]int32, windowSize), ptr: int16(windowSize), mPtr: int16(windowSize / 2), sampleSize: int16(sampleSize), windowSize: int16(windowSize), Ready: false}
	return avg
}

func (a *Avg) ResetCounter() {
	a.Counter = 0
	a.Ready = false
}

func (a *Avg) Add(val int32) {
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
		for p, q := i, i + 1; q < a.windowSize; p, q = p + 1, q + 1 {
			if a.sorted[p] > a.sorted[q] {
				tmp := a.sorted[p]
				a.sorted[p] = a.sorted[q]
				a.sorted[q] = tmp
			} else {
				return
			}
		}
	} else {
		for p, q := i - 1, i; q > 0; p, q = p - 1, q - 1 {
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

func (a *Avg) Median() int32 {
	return a.sorted[a.mPtr]
}

func (a *Avg) Average() int32 {
	left := a.mPtr - a.sampleSize / 2
	right := left + a.sampleSize

	var sum int64 = 0

	for _, x := range a.sorted[left:right] {
		sum += int64(x)
	}
	return int32(sum / int64(a.sampleSize))
}

func main() {
	if err := embd.InitGPIO(); err != nil {
		panic(err)
	}
	defer embd.CloseGPIO()

	sck, err := embd.NewDigitalPin(22)
	if err != nil {
		panic(err)
	}
	defer sck.Close()
	dt, err := embd.NewDigitalPin(27)
	if err != nil {
		panic(err)
	}
	defer dt.Close()

	go func() {
		sigchan := make(chan os.Signal, 10)
		signal.Notify(sigchan)
		<-sigchan
		sck.Close()
		dt.Close()
		os.Exit(0)
	}()

	if err := sck.SetDirection(embd.Out); err != nil {
		panic(err)
	}
	if err := dt.SetDirection(embd.In); err != nil {
		panic(err)
	}

	sck.Write(embd.Low)

	a := NewAvg(20, 5)
	b := NewAvg(20, 5)

	cal := make(chan float64)
	go func(ch chan float64) {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, _ := reader.ReadString('\n')
			text = text[:len(text)-1]
			f, _ := strconv.ParseFloat(text, 64)
			fmt.Printf("Calibrating @%v (%v)\n", f, text)
			ch <- f
		}
	}(cal)
	var cz, cv int32 = 0, 0
	var cm float64 = 0
	for {
		select {
		case x := <- cal:
			if math.Abs(x) < 0.01 {
				cz = b.Median()
			} else {
				cm = x
				cv = b.Median()
			}
		default:
			for ready := false; !ready; {
				if x, err := dt.Read(); err != nil {
					panic(err)
				} else if x == embd.Low {
					ready = true
				}
			}
			var res int32 = 0
			for bit := 0; bit < 24; bit++ {
				res <<= 1
				sck.Write(embd.High)
				if x, err := dt.Read(); err != nil {
					panic(err)
				} else if x == embd.High {
					res |= 1
				}
				sck.Write(embd.Low)
			}
			sck.Write(embd.High)
			sck.Write(embd.Low)
			res <<= 8
			res >>= 8
			if res != -1 {
				a.Add(res)
				if a.Ready {
					b.Add(a.Median())
					a.ResetCounter()
					if b.Ready {
						fmt.Printf("%20.5f %10d\n", cm / float64(cv - cz) * float64(b.Median() - cz), b.Median())
					}
				}
			}
		}
	}
}