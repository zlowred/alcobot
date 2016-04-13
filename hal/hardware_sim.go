// +build !linux !arm

package hal

import (
	"math/rand"
	"time"

	"math"

	"github.com/zlowred/alcobot/hub"
)

type Hal struct {
}

const delay = time.Millisecond * 50

func New(hub *hub.Hub) *Hal {
	go npaPoller(hub)
	go dsPoller(hub)
	go adsPoller(hub)
	go pcaUpdater(hub)
	return &Hal{}
}

func ListW1Devices() []string {
	return []string{"28-Chupacabra", "28-011572120bff", "28-Chtulhu"}
}

func (h *Hal) ResetI2C() {
}

func npaPoller(hub *hub.Hub) {
	x := 0.
	for {
		select {
		case <-hub.Quit:
			return
		default:
			hub.NpaPressureSensor.Send(int16(math.Abs(math.Cos(x) * 15000)))
			hub.NpaTemperatureSensor.Send(int16(rand.Int()))
			x += .017
			time.Sleep(delay)
		}
	}
}

func adsPoller(hub *hub.Hub) {
	x := 0.
	for {
		select {
		case <-hub.Quit:
			return
		default:
			hub.AdsValueSensor.Send(int16(math.Sin(x) * 1000))
			x += .02
			time.Sleep(delay)
		}
	}
}

func dsPoller(hub *hub.Hub) {
	x := 3000
	d := 10
	for {
		select {
		case <-hub.Quit:
			return
		default:
			x += d
			if x > 5000 {
				d = -10
			} else if x < 3000 {
				d = 10
			}
			//hub.DsTemperatureSensor.Send(int16(x))
			hub.DsTemperatureSensor.Send(int16(338)) //=70ºF
			time.Sleep(delay)
		}
	}
}

func pcaUpdater(h *hub.Hub) {
	pwmc := hub.JoinPwmValueGroup(h.PwmOutput)
	for {
		select {
		case <-pwmc:
		case <-h.Quit:
			return
		default:
			time.Sleep(delay)
		}
	}
}
