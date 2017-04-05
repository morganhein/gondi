package main

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/morganhein/gondi"
	"github.com/morganhein/gondi/logger"
	"github.com/morganhein/gondi/schema"
)

func main() {
	log := logger.Log

	cwd, _ := os.Getwd()
	log.Debugf("Current %s\n", cwd)
	csvfile, err := os.Open("devices.csv")
	if err != nil {
		log.Criticalf("Unable to open the devices.csv file: %s. No devices loaded, exiting.", err.Error())
		os.Exit(1)
	}
	defer csvfile.Close()
	options := csv.NewReader(csvfile)

	g := gondi.NewG()
	rows, err := options.ReadAll()
	if err != nil {
		log.Criticalf("Cannot load devices from csv file: %s. No devices loaded, exiting.", err.Error())
		os.Exit(1)
	}

	for _, row := range rows {
		log.Info(row)
		d, err := strconv.Atoi(row[1])
		if err != nil {
			log.Warningf("Error converting devicetype to an integer: %s. Skipping.", err)
			continue
		}
		t, err := strconv.Atoi(row[2])
		if err != nil {
			log.Warningf("Error converting the method type to an integer: %s. Skipping.", err)
			continue
		}
		p, err := strconv.Atoi(row[4])
		if err != nil {
			log.Warningf("Error converting the port to an integer: %s. Skipping.", err)
			continue
		}
		opt := schema.ConnectOptions{
			Host:           row[3],
			Port:           p,
			Username:       row[5],
			Password:       row[6],
			EnablePassword: row[7],
		}
		dev, err := g.Connect(schema.DeviceType(d), row[0], schema.ConnectionMethod(t), opt)

		if err != nil {
			log.Warningf("Cannot connect to device due to: %s. Skipping.", err.Error())
			continue
		}

		log.Debug("Successfully connected to device.")
		time.Sleep(time.Duration(1) * time.Second)
		ret, err := dev.WriteCapture("show run")
		if err != nil {
			log.Warningf("%s\n", err.Error())
			continue
		}
		b, _ := json.MarshalIndent(ret, "", "  ")
		log.Info("\nResult: ", string(b))
		dev.Disconnect()
	}
	g.Shutdown()
}
