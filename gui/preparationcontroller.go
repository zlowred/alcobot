package gui

import (
	"time"

	"fmt"

	"math"

	"github.com/zlowred/goqt/ui"
	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/conv"
	"github.com/zlowred/alcobot/hub"
)

type PreparationController struct {
	screen *RootScreen

	prepare *ui.QPushButton
	pitch   *ui.QPushButton
	minus   *ui.QPushButton
	plus    *ui.QPushButton

	fermenterTemp *ui.QLabel
	targetTemp    *ui.QLabel
	sg            *ui.QLabel
	pidOutput     *ui.QLabel
	output        *ui.QLabel

	sgValue     float64
	dsTemp      int16
	pidOutValue float64
	outValue    float64

	conf *config.Configuration
}

func NewPreparationController(screen *RootScreen) *PreparationController {
	ctl := &PreparationController{screen: screen}

	ctl.prepare = ui.NewPushButtonFromDriver(screen.FindChild("startPreparation"))
	ctl.prepare.OnClicked(func() {
		if ctl.conf == nil {
			return
		}
		ctl.conf.BrewingStartTime = time.Now()
		ctl.conf.Stage = config.PREPARATION
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})

	ctl.pitch = ui.NewPushButtonFromDriver(screen.FindChild("pitchYeast"))
	ctl.pitch.OnClicked(func() {
		if ctl.conf == nil {
			return
		}
		ctl.conf.PitchTime = time.Now()
		ctl.conf.Stage = config.BREWING
		ctl.screen.hub.Configuration.Send(ctl.conf)
		ctl.screen.hub.ScreenChange.Send(config.BREWING_SCREEN)
	})

	ctl.minus = ui.NewPushButtonFromDriver(screen.FindChild("preparationMinus"))
	ctl.minus.OnClicked(func() {
		if ctl.conf == nil {
			return
		}
		if ctl.conf.TemperatureScale == config.F {
			ctl.conf.TargetTemperature = conv.FtoC(conv.CtoF(ctl.conf.TargetTemperature) - 0.1)
		} else {
			ctl.conf.TargetTemperature -= 0.1
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})

	ctl.plus = ui.NewPushButtonFromDriver(screen.FindChild("preparationPlus"))
	ctl.plus.OnClicked(func() {
		if ctl.conf == nil {
			return
		}
		if ctl.conf.TemperatureScale == config.F {
			ctl.conf.TargetTemperature = conv.FtoC(conv.CtoF(ctl.conf.TargetTemperature) + 0.1)
		} else {
			ctl.conf.TargetTemperature += 0.1
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})

	ctl.fermenterTemp = ui.NewLabelFromDriver(screen.FindChild("prepFermenterTemp"))
	ctl.targetTemp = ui.NewLabelFromDriver(screen.FindChild("prepTargetTemp"))
	ctl.sg = ui.NewLabelFromDriver(screen.FindChild("prepSg"))
	ctl.pidOutput = ui.NewLabelFromDriver(screen.FindChild("prepPidOutput"))
	ctl.output = ui.NewLabelFromDriver(screen.FindChild("prepOutput"))

	go ctl.loop()

	return ctl
}

func (ctl *PreparationController) loop() {
	configCh := hub.JoinConfigGroup(ctl.screen.hub.Configuration)
	npaPressureFiltered := hub.JoinInt16Group(ctl.screen.hub.NpaPressureFiltered)
	dsTemperatureFiltered := hub.JoinInt16Group(ctl.screen.hub.DsTemperatureFiltered)
	pid := hub.JoinFloat64Group(ctl.screen.hub.PidOutput)
	pidAdj := hub.JoinFloat64Group(ctl.screen.hub.AdjustedPidOutput)

	for {
		select {
		case <-ctl.screen.hub.Quit:
			return
		case x := <-configCh:
			ctl.conf = x
			ui.Async(func() {
				ctl.plus.SetEnabled(ctl.conf.Stage == config.PREPARATION || ctl.conf.Stage == config.SETUP)
				ctl.minus.SetEnabled(ctl.conf.Stage == config.PREPARATION || ctl.conf.Stage == config.SETUP)
				ctl.prepare.SetEnabled(ctl.conf.Stage == config.SETUP)
				ctl.pitch.SetEnabled(ctl.conf.Stage == config.PREPARATION && math.Abs(conv.DsToC(ctl.dsTemp)-ctl.conf.TargetTemperature) < 0.2)
				if ctl.conf.TemperatureScale == config.F {
					ctl.targetTemp.SetText(fmt.Sprintf("<font color='#00f'>%.1fºF</font>", conv.CtoF(ctl.conf.TargetTemperature)))
					ctl.fermenterTemp.SetText(fmt.Sprintf("Fermenter temp: <font color='#0ff'>%.1fºF</font>", conv.DsToF(ctl.dsTemp)))
				} else {
					ctl.targetTemp.SetText(fmt.Sprintf("<font color='#00f'>%.1fºC</font>", ctl.conf.TargetTemperature))
					ctl.fermenterTemp.SetText(fmt.Sprintf("Fermenter temp: <font color='#0ff'>%.1fºC</font>", conv.DsToC(ctl.dsTemp)))
				}
			})
		case x := <-pid:
			if ctl.conf == nil {
				continue
			}
			ctl.pidOutValue = x / 2.55
			ui.Async(func() {
				ctl.pidOutput.SetText(fmt.Sprintf("PID Output: <font color='#ff0'>%.0f%%</font>", ctl.pidOutValue))
			})
		case x := <-pidAdj:
			if ctl.conf == nil {
				continue
			}
			ctl.outValue = x / 2.55
			ui.Async(func() {
				ctl.output.SetText(fmt.Sprintf("Output: <font color='#f00'>%.0f%%</font>", ctl.outValue))
			})
		case x := <-npaPressureFiltered:
			if ctl.conf == nil {
				continue
			}
			pa := conv.NpaToPa(x, ctl.conf.NpaZero, ctl.conf.NpaMinValue, ctl.conf.NpaMaxValue, ctl.conf.NpaMinPressure, ctl.conf.NpaMaxPressure)
			ctl.sgValue = conv.PaToSg(pa, ctl.conf.NpaCalibration)
			ui.Async(func() {
				ctl.sg.SetText(fmt.Sprintf("SG: %.3f", ctl.sgValue))
			})
		case x := <-dsTemperatureFiltered:
			if ctl.conf == nil {
				continue
			}
			ctl.dsTemp = x
			ui.Async(func() {
				if ctl.conf.TemperatureScale == config.F {
					ctl.fermenterTemp.SetText(fmt.Sprintf("Fermenter temp: <font color='#0ff'>%.1fºF</font>", conv.DsToF(x)))
				} else {
					ctl.fermenterTemp.SetText(fmt.Sprintf("Fermenter temp: <font color='#0ff'>%.1fºC</font>", conv.DsToC(x)))
				}
				ctl.pitch.SetEnabled(ctl.conf.Stage == config.PREPARATION && math.Abs(conv.DsToC(ctl.dsTemp)-ctl.conf.TargetTemperature) < 0.2)
			})

		}
	}
}
