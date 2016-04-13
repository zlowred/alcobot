package heatpump

import (
	"math"
	"time"

	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/hub"
)

type HeatPump struct {
	hub     *hub.Hub
	conf    *config.Configuration
	target  float64
	current float64
	timer   time.Time
	enabled bool
}

func New(h *hub.Hub) *HeatPump {
	heatPump := &HeatPump{hub: h, timer: time.Now(), current: 0, target: 0, enabled: false}
	go heatPump.loop()
	return heatPump
}

func (p *HeatPump) setPwmChannel(posChannel uint8, negChannel uint8, threshold int, min byte, max byte, value float64) {
	if math.Abs(value) < float64(threshold) {
		if posChannel != 255 {
			p.hub.PwmOutput.Send(hub.PwmValue{Channel: posChannel, Value: 0})
		}
		if negChannel != 255 {
			p.hub.PwmOutput.Send(hub.PwmValue{Channel: negChannel, Value: 0})
		}
	} else if value < 0 {
		if posChannel != 255 {
			p.hub.PwmOutput.Send(hub.PwmValue{Channel: posChannel, Value: 0})
		}
		if negChannel != 255 {
			p.hub.PwmOutput.Send(hub.PwmValue{Channel: negChannel, Value: byte(float64(min) + float64(max-min)/255*-value)})
		}
	} else if value > 0 {
		if posChannel != 255 {
			p.hub.PwmOutput.Send(hub.PwmValue{Channel: posChannel, Value: byte(float64(min) + float64(max-min)/255*value)})
		}
		if negChannel != 255 {
			p.hub.PwmOutput.Send(hub.PwmValue{Channel: negChannel, Value: 0})
		}
	}
}

func (p *HeatPump) setPwm() {
	if p.enabled {
		p.hub.AdjustedPidOutput.Send(p.current)
		p.setPwmChannel(0, 1, p.conf.Tec1Threshold, p.conf.Tec1Min, p.conf.Tec1Max, p.current)
		p.setPwmChannel(2, 3, p.conf.Tec2Threshold, p.conf.Tec2Min, p.conf.Tec2Max, p.current)
		p.setPwmChannel(4, 255, p.conf.Fan1Threshold, p.conf.Fan1Min, p.conf.Fan1Max, math.Abs(p.current))
		p.setPwmChannel(5, 255, p.conf.Fan2Threshold, p.conf.Fan2Min, p.conf.Fan2Max, math.Abs(p.current))
		p.setPwmChannel(6, 255, p.conf.Pump1Threshold, p.conf.Pump1Min, p.conf.Pump1Max, math.Abs(p.current))
		p.setPwmChannel(7, 255, p.conf.Pump2Threshold, p.conf.Pump2Min, p.conf.Pump2Max, math.Abs(p.current))
		p.setPwmChannel(15, 255, 0, 0, 255, 255)
	}
}

func (p *HeatPump) loop() {
	pidOutputCh := hub.JoinFloat64Group(p.hub.PidOutput)
	configCh := hub.JoinConfigGroup(p.hub.Configuration)
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			if p.conf == nil {
				continue
			}
			if math.Abs(p.current-p.target) < p.conf.PidSlope {
				p.current = p.target
			} else if p.current < p.target {
				p.current += p.conf.PidSlope
			} else {
				p.current -= p.conf.PidSlope
			}
			p.setPwm()
		case c := <-configCh:
			p.conf = c
		case v := <-pidOutputCh:
			p.enabled = true
			p.target = v
		case <-p.hub.Quit:
			return
		}
	}
}
