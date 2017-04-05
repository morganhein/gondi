package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/morganhein/gondi"
	"github.com/morganhein/gondi/schema"
)

func main() {
	cwd, _ := os.Getwd()
	fmt.Printf("Current %s\n", cwd)
	csvfile, err := os.Open("devices.csv")
	if err != nil {
		log.Panicf("Unable to open the devices.csv file: %s", err.Error())
	}
	defer csvfile.Close()
	options := csv.NewReader(csvfile)

	g := gondi.NewG()
	rows, err := options.ReadAll()
	if err != nil {
		fmt.Printf("Cannot load devices from csv file: %s", err.Error())
	}

	for _, row := range rows {
		fmt.Println(row)
		d, err := strconv.Atoi(row[1])
		if err != nil {
			fmt.Printf("Error converting devicetype to an integer: %s", err)
		}
		fmt.Println(schema.DeviceType(d))
		t, err := strconv.Atoi(row[2])
		if err != nil {
			fmt.Printf("Error converting the method type to an integer: %s", err)
		}
		p, err := strconv.Atoi(row[4])
		if err != nil {
			fmt.Printf("Error converting the port to an integer: %s", err)
		}
		opt := schema.ConnectOptions{
			Host:           row[3],
			Port:           p,
			Username:       row[5],
			Password:       row[6],
			EnablePassword: row[7],
		}
		fmt.Printf("%s\n", opt)
		dev, err := g.Connect(schema.DeviceType(d), row[0], schema.ConnectionMethod(t), opt)

		if err != nil {
			fmt.Printf("Cannot connect to device due to: %s", err.Error())
			os.Exit(1)
		}

		fmt.Println("Successfully connected to device.")
		time.Sleep(time.Duration(1) * time.Second)
		ret, err := dev.WriteCapture("show run")
		fmt.Println("\n\nResult:")
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
		b, _ := json.MarshalIndent(ret, "", "  ")
		println(string(b))
		dev.Disconnect()
	}
	g.Shutdown()
}
