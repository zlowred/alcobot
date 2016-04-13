package series

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptySizeZero(t *testing.T) {
	series := NewSeries(10)
	assert.Zero(t, series.Size())
}

func TestEmptySizeOne(t *testing.T) {
	series := NewSeries(10)
	series.Push(1.)
	assert.Equal(t, 1, series.Size())
}

func TestEmptySizeOverflow(t *testing.T) {
	series := NewSeries(10)
	for i := 0; i < 15; i++ {
		series.Push(float64(i))
	}
	assert.Equal(t, 10, series.Size())
}

func TestNormal(t *testing.T) {
	series := NewSeries(3)
	for i := 0; i < 3; i++ {
		series.Push(float64(i))
	}
	assert.Equal(t, []float64{0., 1., 2.}, series.Get())
}

func TestOverflow(t *testing.T) {
	series := NewSeries(3)
	for i := 0; i < 6; i++ {
		series.Push(float64(i))
	}
	assert.Equal(t, []float64{0., 2.5, 5.}, series.Get())
}

func TestUnderflow(t *testing.T) {
	series := NewSeries(3)
	for i := 0; i < 2; i++ {
		series.Push(float64(i))
	}
	assert.Equal(t, []float64{0., 1.}, series.Get())
}
