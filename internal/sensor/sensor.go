package sensor

import (
	"fmt"
	"log"
	"sync"

	"github.com/d2r2/go-bsbmp"
	"github.com/d2r2/go-i2c"
	"github.com/d2r2/go-logger"
)

var PiSensor *Sensor

// SensorStore is a simple in-memory database of tasks; SensorStore methods are
// safe to call concurrently.
type SensorStore struct {
	sync.Mutex

	sensors map[int]*Sensor
}

func NewSensorStore() *SensorStore {
	initializePiSensor()
	ss := &SensorStore{}
	ss.sensors = make(map[int]*Sensor)
	ss.sensors[int(*PiSensor.SensorID)] = PiSensor
	return ss
}

// Sensor is the retreived environment properties.
type Sensor struct {
	*bsbmp.BMP
	SensorID    *uint8   `json:"sensor_id,omitempty"`
	Temperature *float32 `json:"temperature,omitempty"`
	Humidity    *float32 `json:"humidity,omitempty"`
	Pressue     *float32 `json:"pressure,omitempty"`
	Altitude    *float32 `json:"altitude,omitempty"`
}

// SensorStore retrieves a task from the store, by id. If no such id exists, an
// error is returned.
func (ss *SensorStore) GetSensor(id int) (*Sensor, error) {
	ss.Lock()
	defer ss.Unlock()

	s, ok := ss.sensors[id]
	if ok {
		return s, s.getEnvironment()
	} else {
		return nil, fmt.Errorf("sensor with id=%d not found", id)
	}
}

// GetAllSensors returns all the tasks in the store, in arbitrary order.
func (ss *SensorStore) GetAllSensors() ([]Sensor, error) {
	ss.Lock()
	defer ss.Unlock()

	allSensors := make([]Sensor, 0, len(ss.sensors))
	for _, sensor := range ss.sensors {
		if err := sensor.getEnvironment(); err != nil {
			return nil, fmt.Errorf("error with sensor %v: %v", sensor.SensorID, err)
		}
		allSensors = append(allSensors, *sensor)
	}
	return allSensors, nil
}

func (s *Sensor) getEnvironment() error {

	id, err := s.ReadSensorID()
	if err != nil {
		return fmt.Errorf("read sensor id error: %v", err)
	}
	s.SensorID = &id
	log.Printf("Sensor ID = %v\n", id)

	// Read temperature in celsius degree
	t, err := s.ReadTemperatureC(bsbmp.ACCURACY_HIGH)
	if err != nil {
		return fmt.Errorf("read temperature error: %v", err)
	}
	s.Temperature = &t
	log.Printf("Temprature = %v*C\n", t)
	// Read atmospheric pressure in pascal
	p, err := s.ReadPressurePa(bsbmp.ACCURACY_HIGH)
	if err != nil {
		return fmt.Errorf("read pressure (pascal) error: %v", err)
	}
	s.Pressue = &p
	log.Printf("Pressure = %v Pa\n", p)
	// Read relative humidity in %RH
	supported, rh, err := s.ReadHumidityRH(bsbmp.ACCURACY_HIGH)
	if err != nil {
		return fmt.Errorf("read humidity (%%rh) error: %v", err)
	}
	s.Humidity = &rh
	if !supported {
		log.Printf("Sensor does not support relative humidity")
		s.Humidity = nil
	}
	log.Printf("Relative Humidity = %v %%RH\n", rh)
	// Read atmospheric altitude in meters above sea level, if we assume
	// that pressure at see level is equal to 101325 Pa.
	a, err := s.ReadAltitude(bsbmp.ACCURACY_HIGH)
	if err != nil {
		return fmt.Errorf("read altitude error: %v", err)
	}
	s.Altitude = &a
	log.Printf("Altitude = %v m\n", a)

	return nil
}

func initializePiSensor() {
	// Create new connection to i2c-bus on 1 line with address 0x77.
	// Use i2cdetect utility to find device address over the i2c-bus
	i2c, err := i2c.NewI2C(0x77, 1)
	if err != nil {
		log.Printf("new_i2c error: %v", err)
		return
	}
	if err := logger.ChangePackageLogLevel("i2c", logger.InfoLevel); err != nil {
		log.Printf("error changing package logging: %v", err)
		return
	}

	sense, err := bsbmp.NewBMP(bsbmp.BME280, i2c)
	if err != nil {
		log.Printf("new_bmp error: %v", err)
		return
	}
	if err := logger.ChangePackageLogLevel("bsbmp", logger.InfoLevel); err != nil {
		log.Printf("error changing package logging: %v", err)
		return
	}

	PiSensor = &Sensor{sense, nil, nil, nil, nil, nil}
	if err := PiSensor.getEnvironment(); err != nil {
		log.Printf("failed to initialize PiSensor: %v", err)
		return
	}
}
