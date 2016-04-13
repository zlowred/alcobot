package pid

import (
	"errors"
	"time"

	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/conv"
	"github.com/zlowred/alcobot/hub"
)

type PID struct {
	kP float64
	kI float64
	kD float64

	Input  float64
	Target float64
	Output float64

	bottomLimit float64
	topLimit    float64

	tick  time.Time
	iTerm float64
	input float64

	enabled     bool
	initialized bool

	hub *hub.Hub

	conf *config.Configuration
}

func New(kP float64, kI float64, kD float64, bottomLimit float64, topLimit float64, hub *hub.Hub) (*PID, error) {
	pid := &PID{enabled: false, hub: hub}

	if err := pid.SetTunings(kP, kI, kD); err != nil {
		return nil, err
	}
	if err := pid.SetLimits(bottomLimit, topLimit); err != nil {
		return nil, err
	}

	go pid.loop()

	return pid, nil
}

func (p *PID) Enable() {
	if !p.enabled {
		p.enabled = true
		p.tick = time.Now()
	}
}

func (p *PID) Update() bool {
	if !p.enabled {
		return false
	}

	elapsed := time.Since(p.tick).Nanoseconds()

	err := p.Target - p.Input
	p.iTerm += p.kI * err * float64(elapsed/1000000000)

	if p.iTerm > p.topLimit {
		p.iTerm = p.topLimit
	} else if p.iTerm < p.bottomLimit {
		p.iTerm = p.bottomLimit
	}

	dInput := p.Input - p.input

	output := p.kP*err + p.iTerm - p.kD*dInput/float64(elapsed/1000000000)

	if output > p.topLimit {
		output = p.topLimit
	} else if output < p.bottomLimit {
		output = p.bottomLimit
	}

	wasInitialized := p.initialized
	if !p.initialized {
		p.iTerm = 0
	}
	p.initialized = true

	p.Output = output
	p.input = p.Input
	p.tick = time.Now()
	return wasInitialized
}

func (p *PID) SetLimits(bottomLimit float64, topLimit float64) error {
	if topLimit <= bottomLimit {
		return errors.New("top limit should be above bottom limit")
	}
	p.topLimit, p.bottomLimit = topLimit, bottomLimit
	return nil
}

func (p *PID) SetTunings(kP float64, kI float64, kD float64) error {
	if kP < 0 || kI < 0 || kD < 0 {
		return errors.New("all kP, kI, kD tunings should be positive")
	}
	p.kP, p.kI, p.kD = kP, kI, kD
	return nil
}

func (p *PID) loop() {
	tempCh := hub.JoinInt16Group(p.hub.DsTemperatureFiltered)
	timer := time.NewTimer(time.Second * 1)
	configCh := hub.JoinConfigGroup(p.hub.Configuration)

	for {
		select {
		case val := <-tempCh:
			p.Input = conv.DsToC(val)
		case <-p.hub.Quit:
			return
		case <-timer.C:
			timer = time.NewTimer(time.Second * 1)
			if p.conf == nil {
				break
			}
			if p.conf.Stage != config.BREWING && p.conf.Stage != config.PREPARATION {
				p.hub.PidOutput.Send(0.)
				break
			}
			if p.Update() {
				p.hub.PidOutput.Send(p.Output)
			}
		case x := <-configCh:
			p.conf = x
			p.Target = x.TargetTemperature
		}
	}
}
