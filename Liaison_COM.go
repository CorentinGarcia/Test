package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

const (
	INFLUX_URL     string = "http://localhost:8086"
	INFLUX_DB_NAME string = "ECG_test"
)

func main() {
	port := os.Args[1]
	c := &serial.Config{Name: port, Baud: 115200} // ReadTimeout: time.Millisecond * 10000} Mettre un Readtimeout positif pour un mode non bloquant
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 1024)
	var bufferRX string = ""

	for {
		time.Sleep(10 * time.Millisecond)
		n, err := s.Read((buf))
		fmt.Print("lecture ")
		if err != nil { //Retirer cette partie pour boucler infiniment et se moquer des erreurs
			fmt.Println(err)
			fmt.Println("Erreur Uart RX")
			log.Fatal(err)
		} else {
			bufferRX = bufferRX + hex.EncodeToString(buf[:n])

			//split A55A73
			temp := make([]string, 2048)
			temp = strings.Split(bufferRX, "413535413733")
			fmt.Println(temp)
			fmt.Println(" ")
			//Création du deuxième tableau sans la première trame qui est inutilisable
			temp2 := make([]string, 2048)
			temp2 = temp[1:len(temp)]

			//controle validite trame
			for i := 0; i < len(temp2); i++ {
				decoded, err := hex.DecodeString(temp2[i][0:4])
				if err != nil {
					log.Fatal(err)
				}

				//affiche longueur trame
				x, err := strconv.ParseInt(string(decoded), 16, 64)

				f, err := strconv.ParseFloat(fmt.Sprintf("%d", x), 64)
				ltrame := f
				if err != nil {
					fmt.Println("Problème parser")
					log.Fatal(err)
				}

				if int(ltrame) == (len(temp2[i][4:len(temp2[i])]) / 2) {
					temp3, err := hex.DecodeString(temp2[i][12:20])
					if err != nil {
						log.Fatal(err)
					}

					z, err := strconv.ParseInt(string(temp3), 16, 64)
					g, err := strconv.ParseFloat(fmt.Sprintf("%d", z), 64)
					ltrameuse := g
					if err != nil {
						fmt.Println("Problème Batterie")
						log.Fatal(err)
					}

					dateTime := time.Now()
					dateTimeEpoch := strconv.FormatInt(int64(dateTime.UnixNano()), 10)

					var parametres string
					parametres = "Sensor1"
					parametres = parametres + ","
					parametres = parametres + "Sensor1=Voltage_value"
					parametres = parametres + " "
					parametres = parametres + "value=" + strconv.FormatFloat(ltrameuse, 'f', 2, 64)
					parametres = parametres + " "
					parametres = parametres + dateTimeEpoch

					put(parametres)
					bufferRX = ""
				} else {

				}
			}

		}

	}

}

func put(param string) {

	var url = INFLUX_URL + "/write?db=" + INFLUX_DB_NAME

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(param)))
	req.Header.Set("Content-Type", "data-binary")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
	//body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println("response Body:", string(body))

}

func get() {
	var url = INFLUX_URL + "/query?pretty=true"

	req, err := http.NewRequest("GET", url, nil)

	q := req.URL.Query()
	q.Add("db", INFLUX_DB_NAME)
	q.Add("q", "SELECT * FROM \"Sensor1\"")
	req.URL.RawQuery = q.Encode()

	fmt.Println("url", req.URL)
	fmt.Println("header", req.Header)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

}
