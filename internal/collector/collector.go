package collector

import (
	"time"

	"github.com/marcelb/flowercare-json-exporter/internal/config"
	"github.com/marcelb/flowercare-json-exporter/pkg/miflora"
	"github.com/sirupsen/logrus"
)

// SensorData holds all the information collected from a sensor.
type SensorData struct {
	MACAddress          string  `json:"mac_address"`
	Name                string  `json:"name"`
	Version             string  `json:"version"`
	BatteryPercent      float64 `json:"battery_percent"`
	ConductivitySM      float64 `json:"conductivity_sm"` // Siemens/meter
	BrightnessLux       float64 `json:"brightness_lux"`
	MoisturePercent     float64 `json:"moisture_percent"`
	TemperatureCelsius  float64 `json:"temperature_celsius"`
	LastUpdateTimestamp int64   `json:"last_update_timestamp"`
	Up                  bool    `json:"up"`
}

const (
	// Conversion factor from µS/cm to S/m
	factorConductivity = 0.0001
)

// Flowercare retrieves data from Miflora sensors.
type Flowercare struct {
	Log           logrus.FieldLogger
	Source        func(macAddress string) (miflora.Data, error)
	Sensors       []config.Sensor
	StaleDuration time.Duration
}

// CollectDataAsStructs collects data from all configured sensors and returns it as a slice of SensorData.
func (c *Flowercare) CollectDataAsStructs() []SensorData {
	var results []SensorData

	for _, s := range c.Sensors {
		sensorResult := SensorData{
			MACAddress: s.MacAddress,
			Name:       s.Name,
		}

		data, err := c.Source(s.MacAddress)
		if err != nil {
			c.Log.Errorf("Error getting data for %q: %s", s, err)
			sensorResult.Up = false
			results = append(results, sensorResult)
			continue
		}

		sensorResult.Up = true
		sensorResult.LastUpdateTimestamp = data.Time.Unix()
		sensorResult.Version = data.Firmware.Version

		age := time.Since(data.Time)
		if age >= c.StaleDuration {
			c.Log.Debugf("Data for %q is stale: %s > %s", s, age, c.StaleDuration)
		} else {
			sensorResult.BatteryPercent = float64(data.Firmware.Battery)
			sensorResult.ConductivitySM = float64(data.Sensors.Conductivity) * factorConductivity
			sensorResult.BrightnessLux = float64(data.Sensors.Light)
			sensorResult.MoisturePercent = float64(data.Sensors.Moisture)
			sensorResult.TemperatureCelsius = data.Sensors.Temperature
		}
		results = append(results, sensorResult)
	}

	return results
}
