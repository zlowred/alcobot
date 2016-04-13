// +build linux arm

package hal

import (
	"log"
	"strings"
	"time"

	"github.com/zlowred/embd"
	"github.com/zlowred/embd/controller/pca9955b"
	"github.com/zlowred/embd/convertors/ads1115"
	"github.com/zlowred/embd/sensor/ds18b20"
	"github.com/zlowred/embd/sensor/npa700"

	"github.com/zlowred/alcobot/config"
	"github.com/zlowred/alcobot/hub"
	_ "github.com/zlowred/embd/host/rpi"
)

type Hal struct {
	npaInterrupt chan bool
	dsInterrupt  chan bool
	adsInterrupt chan bool
	pcaInterrupt chan bool

	pwmc <-chan hub.PwmValue

	hub *hub.Hub

	i2c embd.I2CBus
	w1  embd.W1Bus

	conf            *config.Configuration
	fermenterSensor string
}

func ListW1Devices() []string {
	res := make([]string, 0)
	w1 := embd.NewW1Bus(0)

	devs, err := w1.ListDevices()

	if err != nil {
		return res
	}

	for _, dev := range devs {
		if strings.HasPrefix(dev, "28-") {
			res = append(res, dev)
		}
	}
	return res
}

func New(h *hub.Hub) *Hal {
	hal := &Hal{npaInterrupt: make(chan bool, 1), dsInterrupt: make(chan bool, 1), adsInterrupt: make(chan bool, 1), pcaInterrupt: make(chan bool, 1), pwmc: hub.JoinPwmValueGroup(h.PwmOutput), hub: h}
	if err := embd.InitI2C(); err != nil {
		panic(err)
	}
	if err := embd.InitW1(); err != nil {
		panic(err)
	}

	hal.i2c = embd.NewI2CBus(1)
	for i := 0; i < 10; i++ {
		hal.i2c.WriteByte(0, 6)
	}

	hal.w1 = embd.NewW1Bus(0)

	go hal.npaPoller()
	go hal.adsPoller()
	go hal.pcaUpdater()
	go hal.configChange()

	go func() {
		<-hal.hub.Quit
		hal.npaInterrupt <- true
		hal.dsInterrupt <- true
		hal.adsInterrupt <- true
		hal.pcaInterrupt <- true
		embd.CloseI2C()
		embd.CloseW1()

		time.Sleep(time.Second)
		close(hal.npaInterrupt)
		close(hal.adsInterrupt)
		close(hal.dsInterrupt)
		close(hal.pcaInterrupt)
	}()

	return hal
}

func (h *Hal) ResetI2C() {
	for i := 0; i < 10; i++ {
		h.i2c.WriteByte(byte(0), byte(6))
	}
}

func (hal *Hal) configChange() {
	configChangeCh := hub.JoinConfigGroup(hal.hub.Configuration)
	for {
		select {
		case conf := <-configChangeCh:
			if hal.conf != nil && hal.fermenterSensor == conf.FermenterSensor {
				continue
			}
			hal.fermenterSensor = conf.FermenterSensor
			hal.conf = conf
			hal.dsInterrupt <- true
			go hal.dsPoller()
		case <-hal.hub.Quit:
			return
		}

	}
}

func (hal *Hal) npaPoller() {
	sensor := npa700.New(hal.i2c, 0x28)

	for {
		select {
		case <-hal.npaInterrupt:
			return
		default:
			time.Sleep(time.Millisecond * 200)
			if err := sensor.Read(); err != nil {
				log.Printf("NPA error %v", err)
				sensor = npa700.New(hal.i2c, 0x28)
				continue
			}
			hal.hub.NpaTemperatureSensor.Send(sensor.RawTemperature)
			hal.hub.NpaPressureSensor.Send(sensor.RawPressure)
		}
	}
}

func (hal *Hal) adsPoller() {
	sensor := ads1115.New(hal.i2c, 0x48)

	for {
		select {
		case <-hal.dsInterrupt:
			return
		default:
			time.Sleep(time.Millisecond * 50)
			if res, err := sensor.Read(); err != nil {
				log.Printf("ADS error %v", err)
				sensor = ads1115.New(hal.i2c, 0x48)
				continue
			} else {
				hal.hub.AdsValueSensor.Send(int16(res >> 1))
			}
		}
	}
}

func (hal *Hal) dsPoller() {
	log.Println("Starting DS Poller")
	if hal.conf == nil {
		log.Println("No conf")
		return
	}
	if len(hal.conf.FermenterSensor) == 0 {
		log.Println("No sensor")
		return
	}

	w1d, err := hal.w1.Open(hal.conf.FermenterSensor)

	if err != nil {
		log.Printf("W1 device [%v] not found\n", hal.conf.FermenterSensor)
		return
	}
	log.Printf("Usnig W1 device [%v]\n", hal.conf.FermenterSensor)

	sensor := ds18b20.New(w1d)

	//err = sensor.SetResolution(ds18b20.Resolution_12bit)
	//if err != nil {
	//	log.Printf("[%v] set resolution failed: %v\n", hal.conf.FermenterSensor, err)
	//}

	eating := true
	for eating {
		select {
		case <-hal.dsInterrupt:
		default:
			eating = false
		}
	}

	errors := 0
	initialized := false
	for {
		select {
		case <-hal.dsInterrupt:
			return
		default:
			time.Sleep(time.Millisecond * 200)
			err = sensor.ReadTemperature()

			if err != nil || sensor.Raw == int16(-1) {
				errors++
			} else {
				errors = 0
			}
			if errors > 10 {
				errors = 0
				log.Printf("DS error %v", err)
				embd.CloseW1()
				time.Sleep(time.Second)
				hal.w1 = embd.NewW1Bus(0)
				w1d, err = hal.w1.Open(hal.conf.FermenterSensor)

				if err != nil {
					log.Printf("W1 device [%v] not found\n", hal.conf.FermenterSensor)
					return
				}
				sensor = ds18b20.New(w1d)

				err = sensor.SetResolution(ds18b20.Resolution_12bit)
				if err != nil {
					log.Printf("[%v] set resolution failed: %v\n", hal.conf.FermenterSensor, err)
				}
				continue
			}
			if errors == 0 {
				hal.hub.DsTemperatureSensor.Send(sensor.Raw)
				if !initialized {
					for i := 0; i < 100; i++ {
						hal.hub.DsTemperatureSensor.Send(sensor.Raw)
					}
					initialized = true
				}
			}
		}
	}
}

func (hal *Hal) pcaUpdater() {
	pwm := pca9955b.New(hal.i2c, 0x0B)

	for {
		select {
		case <-hal.pcaInterrupt:
			return
		case x := <-hal.pwmc:
			if err := pwm.SetOutput(x.Channel, x.Value); err != nil {
				log.Printf("PCA error %v", err)
				pwm = pca9955b.New(hal.i2c, 0x0B)
				continue
			}
		}
	}
}
