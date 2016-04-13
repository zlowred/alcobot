package config

import "time"

type TempScale int

const (
	F TempScale = iota
	C
)

type Stage int

const (
	SETUP Stage = iota
	PREPARATION
	BREWING
	DONE
)

type Screen int

const (
	SETUP_SCREEN Screen = iota
	PREPARATION_SCREEN
	BREWING_SCREEN
)

type Configuration struct {
	Id                  int
	FermenterSensor     string
	PresenceZero        int16
	PresenceCalibration float64
	PresenceOnTimer     int
	PresenceEnabled     bool
	PresenceTimeout     time.Duration
	TemperatureScale    TempScale
	TargetTemperature   float64
	PidSlope            float64
	NpaZero             int16
	NpaCalibration      float64
	Tec1Threshold       int
	Tec1Min             byte
	Tec1Max             byte
	Tec2Threshold       int
	Tec2Min             byte
	Tec2Max             byte
	Fan1Threshold       int
	Fan1Min             byte
	Fan1Max             byte
	Fan2Threshold       int
	Fan2Min             byte
	Fan2Max             byte
	Pump1Threshold      int
	Pump1Min            byte
	Pump1Max            byte
	Pump2Threshold      int
	Pump2Min            byte
	Pump2Max            byte
	NpaMinValue         float64
	NpaMaxValue         float64
	NpaMinPressure      float64
	NpaMaxPressure      float64
	Stage               Stage
	OG                  float64
	BrewingStartTime    time.Time
	PitchTime           time.Time
}
