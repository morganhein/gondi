package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/morganhein/gondi"
	"github.com/morganhein/gondi/devices"
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
		t, err := strconv.Atoi(row[1])
		if err != nil {
			fmt.Printf("Error converting the method type to an integer: %s", err)
		}
		p, err := strconv.Atoi(row[3])
		if err != nil {
			fmt.Printf("Error converting the port to an integer: %s", err)
		}
		dev, err := g.Connect(devices.Cisco, row[0], byte(t), devices.ConnectOptions{
			Host:           row[2],
			Port:           p,
			Username:       row[4],
			Password:       row[5],
			EnablePassword: row[6],
		})

		if err != nil {
			fmt.Printf("Cannot connect to device due to: %s", err.Error())
			os.Exit(1)
		}

		fmt.Println("Successfully connected to device.")
		dev.Write("show run")
		time.Sleep(10 * time.Second)
		fmt.Println("Exiting.")
		dev.Disconnect()
	}
}
