package gui

import (
	"fmt"
	"time"

	"github.com/zlowred/goqt/ui"
	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/conv"
	"github.com/zlowred/alcobot/hub"
)

type BrewingController struct {
	screen *RootScreen

	temp  *ui.QLabel
	plus  *ui.QPushButton
	minus *ui.QPushButton
	pwm   []*ui.QLabel

	fermenterTemp *ui.QLabel
	npaTemp       *ui.QLabel
	pressure      *ui.QLabel
	sg            *ui.QLabel
	og            *ui.QLabel
	pidOut        *ui.QLabel
	curOut        *ui.QLabel
	adcOut        *ui.QLabel
	timer         *ui.QLabel

	conf      *config.Configuration
	startTime time.Time
}

func NewBrewingController(screen *RootScreen) *BrewingController {
	ctl := &BrewingController{screen: screen, startTime: time.Now()}

	ctl.temp = ui.NewLabelFromDriver(screen.FindChild("targetTemp"))
	ctl.plus = ui.NewPushButtonFromDriver(screen.FindChild("plusButton"))
	ctl.minus = ui.NewPushButtonFromDriver(screen.FindChild("minusButton"))

	ctl.fermenterTemp = ui.NewLabelFromDriver(screen.FindChild("fermenterTemp"))
	ctl.npaTemp = ui.NewLabelFromDriver(screen.FindChild("npaTemp"))
	ctl.pressure = ui.NewLabelFromDriver(screen.FindChild("pressure"))
	ctl.sg = ui.NewLabelFromDriver(screen.FindChild("sg"))
	ctl.og = ui.NewLabelFromDriver(screen.FindChild("og"))
	ctl.pidOut = ui.NewLabelFromDriver(screen.FindChild("pidOut"))
	ctl.curOut = ui.NewLabelFromDriver(screen.FindChild("curOut"))
	ctl.adcOut = ui.NewLabelFromDriver(screen.FindChild("adcOut"))
	ctl.timer = ui.NewLabelFromDriver(screen.FindChild("timer"))

	ctl.pwm = make([]*ui.QLabel, 16)

	for i := 0; i < 16; i++ {
		ctl.pwm[i] = ui.NewLabelFromDriver(screen.FindChild(fmt.Sprintf("pwm%d", i)))
	}

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

	go ctl.loop()

	return ctl
}

func (ctl *BrewingController) loop() {
	configCh := hub.JoinConfigGroup(ctl.screen.hub.Configuration)
	pwmCh := hub.JoinPwmValueGroup(ctl.screen.hub.PwmOutput)
	npaTemperatureFiltered := hub.JoinInt16Group(ctl.screen.hub.NpaTemperatureFiltered)
	npaPressureFiltered := hub.JoinInt16Group(ctl.screen.hub.NpaPressureFiltered)
	dsTemperatureFiltered := hub.JoinInt16Group(ctl.screen.hub.DsTemperatureFiltered)
	adsValue := hub.JoinInt16Group(ctl.screen.hub.AdsValueSensor)
	pid := hub.JoinFloat64Group(ctl.screen.hub.PidOutput)
	pidAdj := hub.JoinFloat64Group(ctl.screen.hub.AdjustedPidOutput)
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			ui.Async(func() {
				now := time.Now().Round(time.Second)
				duration := now.Sub(ctl.startTime.Round(time.Second))
				ctl.timer.SetText(duration.String())
			})
		case <-ctl.screen.hub.Quit:
			return
		case x := <-configCh:
			ctl.conf = x
			ui.Async(func() {
				if ctl.conf.TemperatureScale == config.F {
					ctl.temp.SetText(fmt.Sprintf("%.1fºF", conv.CtoF(ctl.conf.TargetTemperature)))
				} else {
					ctl.temp.SetText(fmt.Sprintf("%.1fºC", ctl.conf.TargetTemperature))
				}
			})
		case x := <-pwmCh:
			if ctl.conf == nil {
				continue
			}
			ui.Async(func() {
				ctl.pwm[x.Channel].SetText(fmt.Sprintf("%d", x.Value))
			})
		case x := <-npaTemperatureFiltered:
			if ctl.conf == nil {
				continue
			}
			ui.Async(func() {
				if ctl.conf.TemperatureScale == config.F {
					ctl.npaTemp.SetText(fmt.Sprintf("%.1fºF", conv.NpaToF(x)))
				} else {
					ctl.npaTemp.SetText(fmt.Sprintf("%.1fºC", conv.NpaToC(x)))
				}
			})
		case x := <-pid:
			if ctl.conf == nil {
				continue
			}
			ui.Async(func() {
				ctl.pidOut.SetText(fmt.Sprintf("%.0f%%", x/2.55))
			})
		case x := <-pidAdj:
			if ctl.conf == nil {
				continue
			}
			ui.Async(func() {
				ctl.curOut.SetText(fmt.Sprintf("%.0f%%", x/2.55))
			})
		case x := <-npaPressureFiltered:
			if ctl.conf == nil {
				continue
			}
			ui.Async(func() {
				pa := conv.NpaToPa(x, ctl.conf.NpaZero, ctl.conf.NpaMinValue, ctl.conf.NpaMaxValue, ctl.conf.NpaMinPressure, ctl.conf.NpaMaxPressure)
				ctl.pressure.SetText(fmt.Sprintf("%.1fPa<font color='cyan'>&nbsp;➟</font>", pa))
				ctl.sg.SetText(fmt.Sprintf("%.4f", conv.PaToSg(pa, ctl.conf.NpaCalibration)))
				ctl.og.SetText(fmt.Sprintf("%.4f", ctl.og))
			})
		case x := <-dsTemperatureFiltered:
			if ctl.conf == nil {
				continue
			}
			ui.Async(func() {
				if ctl.conf.TemperatureScale == config.F {
					ctl.fermenterTemp.SetText(fmt.Sprintf("%.1fºF<font color='#ff0'>&nbsp;➟</font>", conv.DsToF(x)))
				} else {
					ctl.fermenterTemp.SetText(fmt.Sprintf("%.1fºC<font color='#ff0'>&nbsp;➟</font>", conv.DsToC(x)))
				}
			})
		case x := <-adsValue:
			if ctl.conf == nil {
				continue
			}
			ui.Async(func() {
				ctl.adcOut.SetText(fmt.Sprintf("%.d<font color='#0f0'>&nbsp;➟</font>", x))
			})
		}
	}
}
