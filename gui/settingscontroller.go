package gui

import (
	"fmt"
	"time"

	"github.com/zlowred/goqt/ui"
	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/conv"
	"github.com/zlowred/alcobot/hal"
	"github.com/zlowred/alcobot/hub"
)

type SettingsController struct {
	screen *RootScreen

	conf *config.Configuration

	scaleC                   *ui.QPushButton
	scaleF                   *ui.QPushButton
	slopeMinus               *ui.QPushButton
	slope                    *ui.QLabel
	slopePlus                *ui.QPushButton
	fermenterTempSensor      *ui.QComboBox
	npaZero                  *ui.QLabel
	npaZeroBtn               *ui.QPushButton
	npaCalibration           *ui.QLabel
	npaCalibrationBtn        *ui.QPushButton
	presenceZero             *ui.QLabel
	presenceZeroBtn          *ui.QPushButton
	presenceCalibration      *ui.QLabel
	presenceCalibrationMinus *ui.QPushButton
	presenceCalibrationPlus  *ui.QPushButton
	presenceEnabled          *ui.QCheckBox
	tec1ThrMinus             *ui.QPushButton
	tec1Thr                  *ui.QLabel
	tec1ThrPlus              *ui.QPushButton
	tec1MinMinus             *ui.QPushButton
	tec1Min                  *ui.QLabel
	tec1MinPlus              *ui.QPushButton
	tec1MaxMinus             *ui.QPushButton
	tec1Max                  *ui.QLabel
	tec1MaxPlus              *ui.QPushButton
	tec2ThrMinus             *ui.QPushButton
	tec2Thr                  *ui.QLabel
	tec2ThrPlus              *ui.QPushButton
	tec2MinMinus             *ui.QPushButton
	tec2Min                  *ui.QLabel
	tec2MinPlus              *ui.QPushButton
	tec2MaxMinus             *ui.QPushButton
	tec2Max                  *ui.QLabel
	tec2MaxPlus              *ui.QPushButton
	fan1ThrMinus             *ui.QPushButton
	fan1Thr                  *ui.QLabel
	fan1ThrPlus              *ui.QPushButton
	fan1MinMinus             *ui.QPushButton
	fan1Min                  *ui.QLabel
	fan1MinPlus              *ui.QPushButton
	fan1MaxMinus             *ui.QPushButton
	fan1Max                  *ui.QLabel
	fan1MaxPlus              *ui.QPushButton
	fan2ThrMinus             *ui.QPushButton
	fan2Thr                  *ui.QLabel
	fan2ThrPlus              *ui.QPushButton
	fan2MinMinus             *ui.QPushButton
	fan2Min                  *ui.QLabel
	fan2MinPlus              *ui.QPushButton
	fan2MaxMinus             *ui.QPushButton
	fan2Max                  *ui.QLabel
	fan2MaxPlus              *ui.QPushButton
	pump1ThrMinus            *ui.QPushButton
	pump1Thr                 *ui.QLabel
	pump1ThrPlus             *ui.QPushButton
	pump1MinMinus            *ui.QPushButton
	pump1Min                 *ui.QLabel
	pump1MinPlus             *ui.QPushButton
	pump1MaxMinus            *ui.QPushButton
	pump1Max                 *ui.QLabel
	pump1MaxPlus             *ui.QPushButton
	pump2ThrMinus            *ui.QPushButton
	pump2Thr                 *ui.QLabel
	pump2ThrPlus             *ui.QPushButton
	pump2MinMinus            *ui.QPushButton
	pump2Min                 *ui.QLabel
	pump2MinPlus             *ui.QPushButton
	pump2MaxMinus            *ui.QPushButton
	pump2Max                 *ui.QLabel
	pump2MaxPlus             *ui.QPushButton

	dsSensors []string

	zeroTimer          *time.Timer
	zeroCounter        int
	calibrationTimer   *time.Timer
	calibrationCounter int
}

func NewSettingsController(screen *RootScreen) *SettingsController {
	ctl := &SettingsController{screen: screen, dsSensors: make([]string, 0)}

	ctl.bindControls()
	ctl.bindControlListeners()

	go ctl.loop()
	go ctl.updateTempSensors()

	return ctl
}

func eq(a, b []string) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func (ctl *SettingsController) selectFermenterTempSensor() {
	if ctl.conf == nil {
		return
	}
	for i, v := range ctl.dsSensors {
		if ctl.conf.FermenterSensor == v {
			if ctl.fermenterTempSensor.CurrentIndex() != int32(i) {
				ctl.fermenterTempSensor.SetCurrentIndex(int32(i))
			}
			return
		}
	}
	ctl.fermenterTempSensor.SetCurrentIndex(-1)
}

func (ctl *SettingsController) updateTempSensors() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctl.screen.hub.Quit:
			ticker.Stop()
			return
		case <-ticker.C:
			sensors := hal.ListW1Devices()
			if !eq(sensors, ctl.dsSensors) {
				ctl.dsSensors = sensors
				ui.Async(func() {
					for ctl.fermenterTempSensor.Count() > 0 {
						ctl.fermenterTempSensor.RemoveItem(0)
					}
					ctl.fermenterTempSensor.AddItems(sensors)
					ctl.selectFermenterTempSensor()
				})
			}
		}
	}
}

func (ctl *SettingsController) bindControlListeners() {
	ctl.scaleC.OnClicked(func() {
		ctl.conf.TemperatureScale = config.C
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.scaleF.OnClicked(func() {
		ctl.conf.TemperatureScale = config.F
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.slopeMinus.OnClicked(func() {
		if ctl.conf.PidSlope > 2.55 {
			ctl.conf.PidSlope -= 2.55
			ctl.screen.hub.Configuration.Send(ctl.conf)
		}
	})
	ctl.slopePlus.OnClicked(func() {
		if ctl.conf.PidSlope < 255 {
			ctl.conf.PidSlope += 2.55
			ctl.screen.hub.Configuration.Send(ctl.conf)
		}
	})
	skipFirst := true
	ctl.fermenterTempSensor.OnCurrentIndexChanged(func(s string) {
		if ctl.conf == nil {
			return
		}
		if skipFirst {
			skipFirst = false
			return
		}
		if ctl.conf.FermenterSensor != s {
			ctl.conf.FermenterSensor = s
			ctl.screen.hub.Configuration.Send(ctl.conf)
		}
	})
	ctl.npaZeroBtn.OnClicked(func() {
		if ctl.zeroTimer != nil {
			select {
			case <-ctl.zeroTimer.C:
				ctl.zeroTimer = nil
				ctl.zeroCounter = 0
			default:
			}
		}
		if ctl.zeroCounter == 0 {
			ctl.zeroTimer = time.NewTimer(time.Second)
		}
		ctl.zeroCounter++
		if ctl.zeroCounter == 3 {
			if ctl.zeroTimer != nil {
				ctl.zeroTimer.Stop()
				ctl.zeroTimer = nil
				ctl.zeroCounter = 0
			}
			npaPresCh := ctl.screen.hub.NpaPressureSensor.Join()
			x := npaPresCh.Recv().(int16)
			npaPresCh.Close()
			ctl.conf.NpaZero = 8192 - x
			ctl.screen.hub.Configuration.Send(ctl.conf)
		}
	})
	ctl.npaCalibrationBtn.OnClicked(func() {
		if ctl.calibrationTimer != nil {
			select {
			case <-ctl.calibrationTimer.C:
				ctl.calibrationTimer = nil
				ctl.calibrationCounter = 0
			default:
			}
		}
		if ctl.calibrationCounter == 0 {
			ctl.calibrationTimer = time.NewTimer(time.Second)
		}
		ctl.calibrationCounter++
		if ctl.calibrationCounter == 3 {
			if ctl.calibrationTimer != nil {
				ctl.calibrationTimer.Stop()
				ctl.calibrationTimer = nil
				ctl.calibrationCounter = 0
			}
			npaPresCh := ctl.screen.hub.NpaPressureFiltered.Join()
			res := []int16{1, 2, 3, 4, 5}
			ptr := 0
			for !(res[0] == res[1] && res[1] == res[2] && res[2] == res[3] && res[3] == res[4]) {
				res[ptr] = npaPresCh.Recv().(int16)
				ptr = (ptr + 1) % 5
			}
			npaPresCh.Close()
			ctl.conf.NpaCalibration = conv.NpaToPa(res[0], ctl.conf.NpaZero, ctl.conf.NpaMinValue, ctl.conf.NpaMaxValue, ctl.conf.NpaMinPressure, ctl.conf.NpaMaxPressure)
			ctl.screen.hub.Configuration.Send(ctl.conf)
		}
	})
	ctl.presenceZeroBtn.OnClicked(func() {
		adcCh := ctl.screen.hub.AdsValueFiltered.Join()
		x := adcCh.Recv().(int16)
		adcCh.Close()
		ctl.conf.PresenceZero = x
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.presenceCalibrationMinus.OnClicked(func() {
		if ctl.conf.PresenceCalibration > 100 {
			ctl.conf.PresenceCalibration -= 100
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.presenceCalibrationPlus.OnClicked(func() {
		if ctl.conf.PresenceCalibration < 5000 {
			ctl.conf.PresenceCalibration += 100
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.presenceEnabled.OnClicked(func() {
		ctl.conf.PresenceEnabled = ctl.presenceEnabled.IsChecked()
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec1ThrMinus.OnClicked(func() {
		if ctl.conf.Tec1Threshold > 0 {
			ctl.conf.Tec1Threshold--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec1ThrPlus.OnClicked(func() {
		if ctl.conf.Tec1Threshold < 100 {
			ctl.conf.Tec1Threshold++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec1MinMinus.OnClicked(func() {
		if ctl.conf.Tec1Min > 0 {
			ctl.conf.Tec1Min--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec1MinPlus.OnClicked(func() {
		if ctl.conf.Tec1Min < 255 {
			ctl.conf.Tec1Min++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec1MaxMinus.OnClicked(func() {
		if ctl.conf.Tec1Max > 0 {
			ctl.conf.Tec1Max--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec1MaxPlus.OnClicked(func() {
		if ctl.conf.Tec1Max < 255 {
			ctl.conf.Tec1Max++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec2ThrMinus.OnClicked(func() {
		if ctl.conf.Tec2Threshold > 0 {
			ctl.conf.Tec2Threshold--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec2ThrPlus.OnClicked(func() {
		if ctl.conf.Tec2Threshold < 100 {
			ctl.conf.Tec2Threshold++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec2MinMinus.OnClicked(func() {
		if ctl.conf.Tec2Min > 0 {
			ctl.conf.Tec2Min--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec2MinPlus.OnClicked(func() {
		if ctl.conf.Tec2Min < 255 {
			ctl.conf.Tec2Min++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec2MaxMinus.OnClicked(func() {
		if ctl.conf.Tec2Max > 0 {
			ctl.conf.Tec2Max--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.tec2MaxPlus.OnClicked(func() {
		if ctl.conf.Tec2Max < 255 {
			ctl.conf.Tec2Max++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan1ThrMinus.OnClicked(func() {
		if ctl.conf.Fan1Threshold > 0 {
			ctl.conf.Fan1Threshold--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan1ThrPlus.OnClicked(func() {
		if ctl.conf.Fan1Threshold < 100 {
			ctl.conf.Fan1Threshold++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan1MinMinus.OnClicked(func() {
		if ctl.conf.Fan1Min > 0 {
			ctl.conf.Fan1Min--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan1MinPlus.OnClicked(func() {
		if ctl.conf.Fan1Min < 255 {
			ctl.conf.Fan1Min++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan1MaxMinus.OnClicked(func() {
		if ctl.conf.Fan1Max > 0 {
			ctl.conf.Fan1Max--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan1MaxPlus.OnClicked(func() {
		if ctl.conf.Fan1Max < 255 {
			ctl.conf.Fan1Max++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan2ThrMinus.OnClicked(func() {
		if ctl.conf.Fan2Threshold > 0 {
			ctl.conf.Fan2Threshold--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan2ThrPlus.OnClicked(func() {
		if ctl.conf.Fan2Threshold < 100 {
			ctl.conf.Fan2Threshold++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan2MinMinus.OnClicked(func() {
		if ctl.conf.Fan2Min > 0 {
			ctl.conf.Fan2Min--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan2MinPlus.OnClicked(func() {
		if ctl.conf.Fan2Min < 255 {
			ctl.conf.Fan2Min++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan2MaxMinus.OnClicked(func() {
		if ctl.conf.Fan2Max > 0 {
			ctl.conf.Fan2Max--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.fan2MaxPlus.OnClicked(func() {
		if ctl.conf.Fan2Max < 255 {
			ctl.conf.Fan2Max++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump1ThrMinus.OnClicked(func() {
		if ctl.conf.Pump1Threshold > 0 {
			ctl.conf.Pump1Threshold--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump1ThrPlus.OnClicked(func() {
		if ctl.conf.Pump1Threshold < 100 {
			ctl.conf.Pump1Threshold++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump1MinMinus.OnClicked(func() {
		if ctl.conf.Pump1Min > 0 {
			ctl.conf.Pump1Min--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump1MinPlus.OnClicked(func() {
		if ctl.conf.Pump1Min < 255 {
			ctl.conf.Pump1Min++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump1MaxMinus.OnClicked(func() {
		if ctl.conf.Pump1Max > 0 {
			ctl.conf.Pump1Max--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump1MaxPlus.OnClicked(func() {
		if ctl.conf.Pump1Max < 255 {
			ctl.conf.Pump1Max++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump2ThrMinus.OnClicked(func() {
		if ctl.conf.Pump2Threshold > 0 {
			ctl.conf.Pump2Threshold--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump2ThrPlus.OnClicked(func() {
		if ctl.conf.Pump2Threshold < 100 {
			ctl.conf.Pump2Threshold++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump2MinMinus.OnClicked(func() {
		if ctl.conf.Pump2Min > 0 {
			ctl.conf.Pump2Min--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump2MinPlus.OnClicked(func() {
		if ctl.conf.Pump2Min < 255 {
			ctl.conf.Pump2Min++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump2MaxMinus.OnClicked(func() {
		if ctl.conf.Pump2Max > 0 {
			ctl.conf.Pump2Max--
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
	ctl.pump2MaxPlus.OnClicked(func() {
		if ctl.conf.Pump2Max < 255 {
			ctl.conf.Pump2Max++
		}
		ctl.screen.hub.Configuration.Send(ctl.conf)
	})
}

func (ctl *SettingsController) bindControls() {
	ctl.scaleC = ui.NewPushButtonFromDriver(ctl.screen.FindChild("scaleC"))
	ctl.scaleF = ui.NewPushButtonFromDriver(ctl.screen.FindChild("scaleF"))
	ctl.slopeMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("slopeMinus"))
	ctl.slope = ui.NewLabelFromDriver(ctl.screen.FindChild("slopeLabel"))
	ctl.slopePlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("slopePlus"))
	ctl.fermenterTempSensor = ui.NewComboBoxFromDriver(ctl.screen.FindChild("fermTempSensor"))
	ctl.npaZero = ui.NewLabelFromDriver(ctl.screen.FindChild("npaZeroLabel"))
	ctl.npaZeroBtn = ui.NewPushButtonFromDriver(ctl.screen.FindChild("npaZeroButton"))
	ctl.npaCalibration = ui.NewLabelFromDriver(ctl.screen.FindChild("sgCalibrationLabel"))
	ctl.npaCalibrationBtn = ui.NewPushButtonFromDriver(ctl.screen.FindChild("sgCalibrationButton"))
	ctl.presenceZero = ui.NewLabelFromDriver(ctl.screen.FindChild("presenceZeroLabel"))
	ctl.presenceZeroBtn = ui.NewPushButtonFromDriver(ctl.screen.FindChild("presenceZeroButton"))
	ctl.presenceCalibration = ui.NewLabelFromDriver(ctl.screen.FindChild("presenceCalibration"))
	ctl.presenceCalibrationMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("presenceMinus"))
	ctl.presenceCalibrationPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("presencePlus"))
	ctl.presenceEnabled = ui.NewCheckBoxFromDriver(ctl.screen.FindChild("presenceEnabled"))
	ctl.tec1ThrMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec1ThrMinus"))
	ctl.tec1Thr = ui.NewLabelFromDriver(ctl.screen.FindChild("tec1Thr"))
	ctl.tec1ThrPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec1ThrPlus"))
	ctl.tec1MinMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec1MinMinus"))
	ctl.tec1Min = ui.NewLabelFromDriver(ctl.screen.FindChild("tec1Min"))
	ctl.tec1MinPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec1MinPlus"))
	ctl.tec1MaxMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec1MaxMinus"))
	ctl.tec1Max = ui.NewLabelFromDriver(ctl.screen.FindChild("tec1Max"))
	ctl.tec1MaxPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec1MaxPlus"))
	ctl.tec2ThrMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec2ThrMinus"))
	ctl.tec2Thr = ui.NewLabelFromDriver(ctl.screen.FindChild("tec2Thr"))
	ctl.tec2ThrPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec2ThrPlus"))
	ctl.tec2MinMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec2MinMinus"))
	ctl.tec2Min = ui.NewLabelFromDriver(ctl.screen.FindChild("tec2Min"))
	ctl.tec2MinPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec2MinPlus"))
	ctl.tec2MaxMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec2MaxMinus"))
	ctl.tec2Max = ui.NewLabelFromDriver(ctl.screen.FindChild("tec2Max"))
	ctl.tec2MaxPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("tec2MaxPlus"))
	ctl.fan1ThrMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan1ThrMinus"))
	ctl.fan1Thr = ui.NewLabelFromDriver(ctl.screen.FindChild("fan1Thr"))
	ctl.fan1ThrPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan1ThrPlus"))
	ctl.fan1MinMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan1MinMinus"))
	ctl.fan1Min = ui.NewLabelFromDriver(ctl.screen.FindChild("fan1Min"))
	ctl.fan1MinPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan1MinPlus"))
	ctl.fan1MaxMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan1MaxMinus"))
	ctl.fan1Max = ui.NewLabelFromDriver(ctl.screen.FindChild("fan1Max"))
	ctl.fan1MaxPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan1MaxPlus"))
	ctl.fan2ThrMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan2ThrMinus"))
	ctl.fan2Thr = ui.NewLabelFromDriver(ctl.screen.FindChild("fan2Thr"))
	ctl.fan2ThrPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan2ThrPlus"))
	ctl.fan2MinMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan2MinMinus"))
	ctl.fan2Min = ui.NewLabelFromDriver(ctl.screen.FindChild("fan2Min"))
	ctl.fan2MinPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan2MinPlus"))
	ctl.fan2MaxMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan2MaxMinus"))
	ctl.fan2Max = ui.NewLabelFromDriver(ctl.screen.FindChild("fan2Max"))
	ctl.fan2MaxPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("fan2MaxPlus"))
	ctl.pump1ThrMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump1ThrMinus"))
	ctl.pump1Thr = ui.NewLabelFromDriver(ctl.screen.FindChild("pump1Thr"))
	ctl.pump1ThrPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump1ThrPlus"))
	ctl.pump1MinMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump1MinMinus"))
	ctl.pump1Min = ui.NewLabelFromDriver(ctl.screen.FindChild("pump1Min"))
	ctl.pump1MinPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump1MinPlus"))
	ctl.pump1MaxMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump1MaxMinus"))
	ctl.pump1Max = ui.NewLabelFromDriver(ctl.screen.FindChild("pump1Max"))
	ctl.pump1MaxPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump1MaxPlus"))
	ctl.pump2ThrMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump2ThrMinus"))
	ctl.pump2Thr = ui.NewLabelFromDriver(ctl.screen.FindChild("pump2Thr"))
	ctl.pump2ThrPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump2ThrPlus"))
	ctl.pump2MinMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump2MinMinus"))
	ctl.pump2Min = ui.NewLabelFromDriver(ctl.screen.FindChild("pump2Min"))
	ctl.pump2MinPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump2MinPlus"))
	ctl.pump2MaxMinus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump2MaxMinus"))
	ctl.pump2Max = ui.NewLabelFromDriver(ctl.screen.FindChild("pump2Max"))
	ctl.pump2MaxPlus = ui.NewPushButtonFromDriver(ctl.screen.FindChild("pump2MaxPlus"))

}
func (ctl *SettingsController) loop() {
	configCh := hub.JoinConfigGroup(ctl.screen.hub.Configuration)
	for {
		select {
		case <-ctl.screen.hub.Quit:
			return
		case x := <-configCh:
			ctl.conf = x
			ui.Async(func() {
				ctl.selectFermenterTempSensor()
				if x.TemperatureScale == config.F {
					ctl.scaleC.SetChecked(false)
					ctl.scaleF.SetChecked(true)
				} else {
					ctl.scaleF.SetChecked(false)
					ctl.scaleC.SetChecked(true)
				}
				ctl.slope.SetText(fmt.Sprintf("PID Slope:%.0f%%/s", x.PidSlope/2.55))
				ctl.npaZero.SetText(fmt.Sprintf("Zero point: %v", x.NpaZero))
				ctl.npaCalibration.SetText(fmt.Sprintf("Calibration: %.1f", x.NpaCalibration))
				ctl.presenceZero.SetText(fmt.Sprintf("Zero point: %v", x.PresenceZero))
				ctl.presenceCalibration.SetText(fmt.Sprintf("Calibration: %v", x.PresenceCalibration))
				ctl.presenceEnabled.SetChecked(x.PresenceEnabled)
				ctl.tec1Thr.SetText(fmt.Sprintf("Threshold: %v%%", x.Tec1Threshold))
				ctl.tec1Min.SetText(fmt.Sprintf("Min: %v", x.Tec1Min))
				ctl.tec1Max.SetText(fmt.Sprintf("Max: %v", x.Tec1Max))
				ctl.tec2Thr.SetText(fmt.Sprintf("Threshold: %v%%", x.Tec2Threshold))
				ctl.tec2Min.SetText(fmt.Sprintf("Min: %v", x.Tec2Min))
				ctl.tec2Max.SetText(fmt.Sprintf("Max: %v", x.Tec2Max))
				ctl.fan1Thr.SetText(fmt.Sprintf("Threshold: %v%%", x.Fan1Threshold))
				ctl.fan1Min.SetText(fmt.Sprintf("Min: %v", x.Fan1Min))
				ctl.fan1Max.SetText(fmt.Sprintf("Max: %v", x.Fan1Max))
				ctl.fan2Thr.SetText(fmt.Sprintf("Threshold: %v%%", x.Fan2Threshold))
				ctl.fan2Min.SetText(fmt.Sprintf("Min: %v", x.Fan2Min))
				ctl.fan2Max.SetText(fmt.Sprintf("Max: %v", x.Fan2Max))
				ctl.pump1Thr.SetText(fmt.Sprintf("Threshold: %v%%", x.Pump1Threshold))
				ctl.pump1Min.SetText(fmt.Sprintf("Min: %v", x.Pump1Min))
				ctl.pump1Max.SetText(fmt.Sprintf("Max: %v", x.Pump1Max))
				ctl.pump2Thr.SetText(fmt.Sprintf("Threshold: %v%%", x.Pump2Threshold))
				ctl.pump2Min.SetText(fmt.Sprintf("Min: %v", x.Pump2Min))
				ctl.pump2Max.SetText(fmt.Sprintf("Max: %v", x.Pump2Max))

			})
		}
	}
}
