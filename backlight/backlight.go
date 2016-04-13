package backlight

import (
	"log"
	"math"
	"os"
	"time"

	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/hub"
)

type Backlight struct {
	timer    *time.Timer
	count    int
	on       bool
	offTimer bool
	conf     *config.Configuration
}

func New(h *hub.Hub) *Backlight {
	backlight := &Backlight{time.NewTimer(time.Hour), 0, false, false, nil}

	go func() {
		adcValueCh := hub.JoinInt16Group(h.AdsValueSensor)
		configCh := hub.JoinConfigGroup(h.Configuration)

		for {
			select {
			case x := <-configCh:
				backlight.conf = x
			case x := <-adcValueCh:
				if backlight.conf == nil {
					continue
				}
				if math.Abs(float64(x-backlight.conf.PresenceZero)) < backlight.conf.PresenceCalibration {
					backlight.count = 0
					if !backlight.on {
						continue
					}
					if !backlight.offTimer {
						backlight.offTimer = true
						backlight.timer.Reset(backlight.conf.PresenceTimeout)
					}
				} else {
					if !backlight.on {
						backlight.count++
						if backlight.count > backlight.conf.PresenceOnTimer {
							backlight.On()
						}
					} else {
						backlight.offTimer = false
						backlight.timer.Stop()
					}
				}
			case <-h.Quit:
				return
			case <-backlight.timer.C:
				if backlight.conf == nil {
					continue
				}
				backlight.Off()

			}
		}
	}()

	backlight.On()
	return backlight
}

func (b *Backlight) On() {
	b.timer.Stop()
	b.count = 0
	b.offTimer = false
	b.on = true
	backlightState("0")
}

func (b *Backlight) Off() {
	b.timer.Stop()
	b.count = 0
	b.offTimer = false
	b.on = false
	backlightState("1")
}

func backlightState(state string) {
	if f, err := os.OpenFile("/sys/class/backlight/rpi_backlight/bl_power", os.O_RDWR, os.ModeExclusive); err == nil {
		defer f.Close()
		f.WriteString(state)
	} else {
		log.Printf("Can't access backlight interface: %v", err)
	}
}
