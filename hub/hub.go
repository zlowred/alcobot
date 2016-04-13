package hub

import (
	"database/sql"
	"log"
	"time"

	"github.com/IvanMalison/bcast"
	"github.com/eapache/channels"

	"github.com/zlowred/alcobot/avg"
	"github.com/zlowred/alcobot/config"

	"math"

	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type DataPoint struct {
	Id          int
	Step        int
	TargetTemp  float64
	CurrentTemp float64
	SG          float64
	PID         float64
	Power       float64
}

type PwmValue struct {
	Channel uint8
	Value   byte
}

type Hub struct {
	Quit               chan bool
	FlightRecorderLock chan bool

	NpaTemperatureSensor *bcast.Group
	NpaPressureSensor    *bcast.Group
	DsTemperatureSensor  *bcast.Group
	AdsValueSensor       *bcast.Group

	PwmOutput *bcast.Group
	PidOutput *bcast.Group

	NpaTemperatureFiltered *bcast.Group
	NpaPressureFiltered    *bcast.Group
	DsTemperatureFiltered  *bcast.Group
	AdsValueFiltered       *bcast.Group

	Configuration     *bcast.Group
	AdjustedPidOutput *bcast.Group

	ScreenChange *bcast.Group

	DataPoints *bcast.Group

	npaTemperatureFilter *avg.Avg
	npaPressureFilter    *avg.Avg
	dsTemperatureFilter  *avg.Avg
	adsValueFilter       *avg.Avg

	fermenterSensor string

	Conf   *config.Configuration
	db     *sql.DB
	dbLock sync.Mutex
}

func New() (*Hub, error) {

	pwm := bcast.NewGroup()
	hub := &Hub{
		Quit: make(chan bool), PidOutput: bcast.NewGroup(), PwmOutput: pwm, Configuration: bcast.NewGroup(), AdjustedPidOutput: bcast.NewGroup(),
		NpaTemperatureSensor: bcast.NewGroup(), NpaPressureSensor: bcast.NewGroup(), DsTemperatureSensor: bcast.NewGroup(), AdsValueSensor: bcast.NewGroup(),
		NpaTemperatureFiltered: bcast.NewGroup(), NpaPressureFiltered: bcast.NewGroup(), DsTemperatureFiltered: bcast.NewGroup(), AdsValueFiltered: bcast.NewGroup(),
		npaTemperatureFilter: avg.NewAvg(100, 20), npaPressureFilter: avg.NewAvg(100, 20), dsTemperatureFilter: avg.NewAvg(30, 10), adsValueFilter: avg.NewAvg(100, 20),
		ScreenChange: bcast.NewGroup(), FlightRecorderLock: make(chan bool), DataPoints: bcast.NewGroup(),
	}

	db, err := sql.Open("sqlite3", "./alcobot.db")
	if err != nil {
		log.Fatal(err)
	}
	hub.db = db

	hub.queryDb(query("configTableExists.sql"), func(rows *sql.Rows) {
		if rows.Next() {
			return
		}
		hub.execDb(query("createConfigTable.sql"), func(r sql.Result) {
			hub.execDb(query("insertDefaultConfig.sql"), nil)
		})
	})

	hub.queryDb(query("dataTableExists.sql"), func(rows *sql.Rows) {
		if rows.Next() {
			return
		}
		hub.execDb(query("createDataTable.sql"), nil)
	})

	go hub.NpaTemperatureFiltered.Broadcast(0)
	go hub.NpaPressureFiltered.Broadcast(0)
	go hub.DsTemperatureFiltered.Broadcast(0)
	go hub.AdsValueFiltered.Broadcast(0)

	go hub.PidOutput.Broadcast(0)
	go hub.AdjustedPidOutput.Broadcast(0)
	go hub.PwmOutput.Broadcast(0)

	go hub.Configuration.Broadcast(0)
	go hub.AdjustedPidOutput.Broadcast(0)

	go hub.ScreenChange.Broadcast(0)

	go hub.DataPoints.Broadcast(0)

	go hub.NpaTemperatureSensor.Broadcast(0)
	go hub.NpaPressureSensor.Broadcast(0)
	go hub.DsTemperatureSensor.Broadcast(0)
	go hub.AdsValueSensor.Broadcast(0)

	go hub.loop()

	go hub.setup()

	return hub, nil
}

func (h *Hub) loadDataPoints() {
	defer func() {
		h.FlightRecorderLock <- true
	}()

	if h.Conf.Stage < config.PREPARATION {
		log.Printf("Skipping loading saved data points (current stage = %v)\n", h.Conf.Stage)
		return
	}
	log.Println("Loading saved data points")

	lastStep := 0
	h.queryDb(query("selectDataPoints.sql"), func(r *sql.Rows) {
		for r.Next() {
			var Id int
			var Step int
			var TargetTemp sql.NullFloat64
			var CurrentTemp sql.NullFloat64
			var SG sql.NullFloat64
			var PID sql.NullFloat64
			var Power sql.NullFloat64

			r.Scan(
				&Id,
				&Step,
				&TargetTemp,
				&CurrentTemp,
				&SG,
				&PID,
				&Power,
			)

			dp := &DataPoint{Id: Id, Step: Step}
			if TargetTemp.Valid {
				dp.TargetTemp = TargetTemp.Float64
			} else {
				dp.TargetTemp = math.NaN()
			}
			if CurrentTemp.Valid {
				dp.CurrentTemp = CurrentTemp.Float64
			} else {
				dp.CurrentTemp = math.NaN()
			}
			if SG.Valid {
				dp.SG = SG.Float64
			} else {
				dp.SG = math.NaN()
			}
			if PID.Valid {
				dp.PID = PID.Float64
			} else {
				dp.PID = math.NaN()
			}
			if Power.Valid {
				dp.Power = Power.Float64
			} else {
				dp.Power = math.NaN()
			}
			lastStep = dp.Step
			h.DataPoints.Send(dp)
		}
	})

	log.Printf("Completed loading saved %v data points\n", lastStep)

	if h.Conf.Stage == config.DONE {
		return
	}

	points := make([]*DataPoint, 0)
	count := 0
	for lastStep < int((time.Now().Sub(h.Conf.BrewingStartTime) / time.Second)) {
		lastStep++
		tmp := &DataPoint{h.Conf.Id, lastStep, math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN()}
		h.DataPoints.Send(tmp)
		points = append(points, tmp)
		count++
	}
	h.saveDataPoints(points)
	if count > 0 {
		log.Printf("Created %d missing data points\n", count)
	}
}

func (h *Hub) SaveDataPoint(dp *DataPoint) {
	h.dbLock.Lock()
	defer h.dbLock.Unlock()
	tx, err := h.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare(query("insertDataPoint.sql"))
	if err != nil {
		panic(err)
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		h.Conf.Id,
		dp.Step,
		dp.TargetTemp,
		dp.CurrentTemp,
		dp.SG,
		dp.PID,
		dp.Power,
	)

	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
}

func (h *Hub) saveDataPoints(dps []*DataPoint) {
	h.dbLock.Lock()
	defer h.dbLock.Unlock()
	tx, err := h.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare(query("insertDataPoint.sql"))
	if err != nil {
		panic(err)
		log.Fatal(err)
	}
	defer stmt.Close()

	for _, dp := range dps {
		_, err = stmt.Exec(
			h.Conf.Id,
			dp.Step,
			dp.TargetTemp,
			dp.CurrentTemp,
			dp.SG,
			dp.PID,
			dp.Power,
		)
	}

	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
}

func (h *Hub) loop() {
	npaTemperatureCh := JoinInt16Group(h.NpaTemperatureSensor)
	npaPressureCh := JoinInt16Group(h.NpaPressureSensor)
	dsTemperatureCh := JoinInt16Group(h.DsTemperatureSensor)
	adsValueCh := JoinInt16Group(h.AdsValueSensor)
	configCh := JoinConfigGroup(h.Configuration)

	for {
		select {
		case x := <-configCh:
			h.Conf = x
			if h.fermenterSensor != x.FermenterSensor {
				h.fermenterSensor = x.FermenterSensor
				h.dsTemperatureFilter.ResetCounter()
			}
			h.saveConfig()
		case x := <-npaTemperatureCh:
			h.npaTemperatureFilter.Add(x)
			if h.npaTemperatureFilter.Ready {
				h.NpaTemperatureFiltered.Send(h.npaTemperatureFilter.Average())
			}
		case x := <-npaPressureCh:
			h.npaPressureFilter.Add(x)
			if h.npaPressureFilter.Ready {
				h.NpaPressureFiltered.Send(h.npaPressureFilter.Average())
			}
		case x := <-dsTemperatureCh:
			h.dsTemperatureFilter.Add(x)
			if h.dsTemperatureFilter.Ready {
				h.DsTemperatureFiltered.Send(h.dsTemperatureFilter.Average())
			}
		case x := <-adsValueCh:
			h.adsValueFilter.Add(x)
			if h.adsValueFilter.Ready {
				h.AdsValueFiltered.Send(h.adsValueFilter.Average())
			}
		case <-h.Quit:
			h.db.Close()

			h.NpaTemperatureFiltered.Close()
			h.NpaPressureFiltered.Close()
			h.DsTemperatureFiltered.Close()
			h.AdsValueFiltered.Close()

			h.PidOutput.Close()
			h.PwmOutput.Close()

			h.ScreenChange.Close()

			h.DataPoints.Close()

			h.NpaTemperatureSensor.Close()
			h.NpaPressureSensor.Close()
			h.DsTemperatureSensor.Close()
			h.AdsValueSensor.Close()

			h.Configuration.Close()
			h.AdjustedPidOutput.Close()

			return
		}
	}
}

func (h *Hub) setup() {
	time.Sleep(time.Millisecond * 1000)
	h.queryDb(query("selectLatestConfig.sql"), func(r *sql.Rows) {
		for r.Next() {
			conf := &config.Configuration{}
			r.Scan(&conf.Id,
				&conf.FermenterSensor,
				&conf.PresenceZero,
				&conf.PresenceCalibration,
				&conf.PresenceOnTimer,
				&conf.PresenceEnabled,
				&conf.PresenceTimeout,
				&conf.TemperatureScale,
				&conf.TargetTemperature,
				&conf.PidSlope,
				&conf.NpaZero,
				&conf.NpaCalibration,
				&conf.Tec1Threshold,
				&conf.Tec1Min,
				&conf.Tec1Max,
				&conf.Tec2Threshold,
				&conf.Tec2Min,
				&conf.Tec2Max,
				&conf.Fan1Threshold,
				&conf.Fan1Min,
				&conf.Fan1Max,
				&conf.Fan2Threshold,
				&conf.Fan2Min,
				&conf.Fan2Max,
				&conf.Pump1Threshold,
				&conf.Pump1Min,
				&conf.Pump1Max,
				&conf.Pump2Threshold,
				&conf.Pump2Min,
				&conf.Pump2Max,
				&conf.NpaMinValue,
				&conf.NpaMaxValue,
				&conf.NpaMinPressure,
				&conf.NpaMaxPressure,
				&conf.Stage,
				&conf.OG,
				&conf.BrewingStartTime,
				&conf.PitchTime)
			log.Printf("Loaded config: %#v\n", conf)
			h.Configuration.Send(conf)
			switch conf.Stage {
			case config.SETUP:
				h.ScreenChange.Send(config.SETUP_SCREEN)
			case config.PREPARATION:
				h.ScreenChange.Send(config.PREPARATION_SCREEN)
			case config.BREWING:
				h.ScreenChange.Send(config.BREWING_SCREEN)
			}
		}
	})

	time.Sleep(time.Millisecond * 1000)

	h.loadDataPoints()
}
func JoinInt16Group(group *bcast.Group) <-chan int16 {
	ch := make(chan int16)
	channels.Unwrap(channels.Wrap(group.Join().Read), ch)
	return (<-chan int16)(ch)
}

func JoinFloat64Group(group *bcast.Group) <-chan float64 {
	ch := make(chan float64)
	channels.Unwrap(channels.Wrap(group.Join().Read), ch)
	return (<-chan float64)(ch)
}

func JoinPwmValueGroup(group *bcast.Group) <-chan PwmValue {
	ch := make(chan PwmValue)
	channels.Unwrap(channels.Wrap(group.Join().Read), ch)
	return (<-chan PwmValue)(ch)
}

func JoinStringGroup(group *bcast.Group) <-chan string {
	ch := make(chan string)
	channels.Unwrap(channels.Wrap(group.Join().Read), ch)
	return (<-chan string)(ch)
}

func JoinConfigGroup(group *bcast.Group) <-chan *config.Configuration {
	ch := make(chan *config.Configuration)
	channels.Unwrap(channels.Wrap(group.Join().Read), ch)
	return (<-chan *config.Configuration)(ch)
}

func JoinDataPointGroup(group *bcast.Group) <-chan *DataPoint {
	ch := make(chan *DataPoint)
	channels.Unwrap(channels.Wrap(group.Join().Read), ch)
	return (<-chan *DataPoint)(ch)
}

func JoinScreenGroup(group *bcast.Group) <-chan config.Screen {
	ch := make(chan config.Screen)
	channels.Unwrap(channels.Wrap(group.Join().Read), ch)
	return (<-chan config.Screen)(ch)
}

func (hub *Hub) queryDb(stmt string, f func(rows *sql.Rows)) {
	if rows, err := hub.db.Query(stmt); err != nil {
		log.Fatal(err)
	} else {
		defer rows.Close()
		f(rows)
	}
}

func (hub *Hub) execDb(stmt string, f func(rows sql.Result)) {
	if res, err := hub.db.Exec(stmt); err != nil {
		log.Fatal(err)
	} else {
		if f != nil {
			f(res)
		}
	}
}

func query(name string) string {
	if data, err := Asset("sql/" + name); err != nil {
		panic(err)
	} else {
		return string(data)
	}
}

func (h *Hub) saveConfig() {
	h.dbLock.Lock()
	defer h.dbLock.Unlock()
	tx, err := h.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare(query("updateLastConfig.sql"))
	if err != nil {
		panic(err)
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		h.Conf.FermenterSensor,
		h.Conf.PresenceZero,
		h.Conf.PresenceCalibration,
		h.Conf.PresenceOnTimer,
		h.Conf.PresenceEnabled,
		int(h.Conf.PresenceTimeout/time.Second),
		h.Conf.TemperatureScale,
		h.Conf.TargetTemperature,
		h.Conf.PidSlope,
		h.Conf.NpaZero,
		h.Conf.NpaCalibration,
		h.Conf.Tec1Threshold,
		h.Conf.Tec1Min,
		h.Conf.Tec1Max,
		h.Conf.Tec2Threshold,
		h.Conf.Tec2Min,
		h.Conf.Tec2Max,
		h.Conf.Fan1Threshold,
		h.Conf.Fan1Min,
		h.Conf.Fan1Max,
		h.Conf.Fan2Threshold,
		h.Conf.Fan2Min,
		h.Conf.Fan2Max,
		h.Conf.Pump1Threshold,
		h.Conf.Pump1Min,
		h.Conf.Pump1Max,
		h.Conf.Pump2Threshold,
		h.Conf.Pump2Min,
		h.Conf.Pump2Max,
		h.Conf.NpaMinValue,
		h.Conf.NpaMaxValue,
		h.Conf.NpaMinPressure,
		h.Conf.NpaMaxPressure,
		h.Conf.Stage,
		h.Conf.OG,
		h.Conf.BrewingStartTime,
		h.Conf.PitchTime)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
}
