package flightrecorder

import (
	"log"

	"math"
	"time"

	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/conv"
	"github.com/zlowred/alcobot/hub"
)

type FlightRecorder struct {
	hub         *hub.Hub
	step        int
	currentTemp float64
	sg          float64
	pid         float64
	power       float64
	conf        *config.Configuration
}

func New(hub *hub.Hub) {
	r := &FlightRecorder{hub: hub, step: 0, currentTemp: math.NaN(), sg: math.NaN(), pid: math.NaN(), power: math.NaN()}
	go r.loop()
}

func (r *FlightRecorder) loop() {

	configCh := hub.JoinConfigGroup(r.hub.Configuration)
	currentTempCh := hub.JoinInt16Group(r.hub.DsTemperatureFiltered)
	pressureCh := hub.JoinInt16Group(r.hub.NpaPressureFiltered)
	pidCh := hub.JoinFloat64Group(r.hub.PidOutput)
	powerCh := hub.JoinFloat64Group(r.hub.AdjustedPidOutput)
	dpCh := hub.JoinDataPointGroup(r.hub.DataPoints)

	timer := time.NewTimer(time.Second)
	timer.Stop()

	for {
		select {
		case <-r.hub.FlightRecorderLock:
			log.Println("Enabling flight recorder")
			timer.Reset(time.Millisecond * time.Duration(1000-int(time.Now().Nanosecond())/1000000))
		case x := <-dpCh:
			r.step = x.Step
		case <-r.hub.Quit:
			return
		case x := <-configCh:
			r.conf = x
		case x := <-currentTempCh:
			r.currentTemp = float64(x)
		case x := <-pressureCh:
			if r.conf == nil {
				continue
			}
			pres := conv.NpaToPa(x, r.conf.NpaZero, r.conf.NpaMinValue, r.conf.NpaMaxValue, r.conf.NpaMinPressure, r.conf.NpaMaxPressure)
			r.sg = conv.PaToSg(pres, r.conf.NpaCalibration)
		case x := <-pidCh:
			r.pid = x / 2.55
		case x := <-powerCh:
			r.power = x / 2.55
		case <-timer.C:
			if r.conf != nil && r.conf.Stage == config.PREPARATION {
				r.step++
				dp := &hub.DataPoint{r.conf.Id, r.step, r.conf.TargetTemperature, r.currentTemp, math.NaN(), r.pid, r.power}
				r.hub.SaveDataPoint(dp)
				r.hub.DataPoints.Send(dp)
			} else if r.conf != nil && r.conf.Stage == config.BREWING {
				r.step++
				dp := &hub.DataPoint{r.conf.Id, r.step, r.conf.TargetTemperature, r.currentTemp, r.sg, r.pid, r.power}
				r.hub.SaveDataPoint(dp)
				r.hub.DataPoints.Send(dp)
			}
			timer.Reset(time.Millisecond * time.Duration(1000-int(time.Now().Nanosecond())/1000000))
		}
	}
}
