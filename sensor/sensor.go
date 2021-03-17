package sensor

import (
	"fmt"
	"log"

	"github.com/d2r2/go-bsbmp"
	"github.com/d2r2/go-i2c"
	"github.com/d2r2/go-logger"
)

var piSensor *bsbmp.BMP

// Environment is the retreived environment properties.
type Environment struct {
	SensorID    *uint8   `json:"sensor_id,omitempty"`
	Temperature *float32 `json:"temperature,omitempty"`
	Humidity    *float32 `json:"humidity,omitempty"`
	Pressue     *float32 `json:"pressure,omitempty"`
	Altitude    *float32 `json:"altitude,omitempty"`
}

func (e *Environment) getEnvironment() error {

	id, err := piSensor.ReadSensorID()
	if err != nil {
		return fmt.Errorf("read sensor id error: %v", err)
	}
	e.SensorID = &id
	log.Printf("Sensor ID = %v\n", id)

	// Read temperature in celsius degree
	t, err := piSensor.ReadTemperatureC(bsbmp.ACCURACY_HIGH)
	if err != nil {
		return fmt.Errorf("read temperature error: %v", err)
	}
	e.Temperature = &t
	log.Printf("Temprature = %v*C\n", t)
	// Read atmospheric pressure in pascal
	p, err := piSensor.ReadPressurePa(bsbmp.ACCURACY_HIGH)
	if err != nil {
		return fmt.Errorf("read pressure (pascal) error: %v", err)
	}
	e.Pressue = &p
	log.Printf("Pressure = %v Pa\n", p)
	// Read relative humidity in %RH
	supported, rh, err := piSensor.ReadHumidityRH(bsbmp.ACCURACY_HIGH)
	if err != nil {
		return fmt.Errorf("read humidity (%%rh) error: %v", err)
	}
	e.Humidity = &rh
	if !supported {
		log.Printf("Sensor does not support relative humidity")
		e.Humidity = nil
	}
	log.Printf("Relative Humidity = %v %%RH\n", rh)
	// Read atmospheric altitude in meters above sea level, if we assume
	// that pressure at see level is equal to 101325 Pa.
	a, err := piSensor.ReadAltitude(bsbmp.ACCURACY_HIGH)
	if err != nil {
		return fmt.Errorf("read altitude error: %v", err)
	}
	e.Altitude = &a
	log.Printf("Altitude = %v m\n", a)

	return nil
}

func init() {
	// Create new connection to i2c-bus on 1 line with address 0x77.
	// Use i2cdetect utility to find device address over the i2c-bus
	i2c, err := i2c.NewI2C(0x77, 1)
	if err != nil {
		log.Fatalf("new_i2c error: %v", err)
	}
	logger.ChangePackageLogLevel("i2c", logger.InfoLevel)

	piSensor, err = bsbmp.NewBMP(bsbmp.BME280, i2c)
	if err != nil {
		log.Fatalf("new_bmp error: %v", err)
	}
	logger.ChangePackageLogLevel("bsbmp", logger.InfoLevel)

}
