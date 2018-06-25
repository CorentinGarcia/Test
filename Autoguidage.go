package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	INFLUX_URL     string = "http://vps198578.ovh.net:8086"
	INFLUX_DB_NAME string = "autoguidage"

	SLACKWEBHOOK_URL string = "https://hooks.slack.com/services/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"
)

//Influxdb config parameters struct
type influxdbConfigT struct {
	url    string
	dbname string
	login  string
	pwd    string
}

var slackWebHookURL string

var influxdbConfig = influxdbConfigT{

	url:    INFLUX_URL,
	dbname: INFLUX_DB_NAME,
	login:  "",
	pwd:    "",
}

func sendSlack(params string) {

	req, err := http.NewRequest("POST", slackWebHookURL, bytes.NewBuffer([]byte("{\"text\" :\""+params+","+time.Now().String()+"\"}")))
	req.Header.Set("Content-Type", "application/json")

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

func putInflux(param string) {

	//add timestamp
	dateTime := time.Now() //time.Date(y, time.Month(m), d, h, mi, s, 0, time.Local)
	dateTimeEpoch := strconv.Itoa(int(dateTime.UnixNano()))
	//fmt.Println(dateTimeEpoch)

	//var url = INFLUX_URL + "/write?db=" + INFLUX_DB_NAME
	var url = influxdbConfig.url + "/write?&u=" + influxdbConfig.login + "&p=" + influxdbConfig.pwd + "&db=" + influxdbConfig.dbname

	//fmt.Println("Influx request:", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(param+" "+dateTimeEpoch)))
	req.Header.Set("Content-Type", "data-binary")

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

func checkFrameIntegrity(payload string) uint8 {

	if len(payload) > 56 { //todo : remplacer par taille head + taille crc + length value
		fmt.Println("integrity error : too long")
		return 0
	}

	n, err := strconv.ParseUint(payload[22:24], 16, 16) //DEG to bytes

	if err != nil {
		return 0
	}

	if len(payload) != (int)(n*2+28) {
		fmt.Println("integrity error : length error")
		return 0
	}

	//TODO : check CRC

	return 1
}

func parseGpsLat(payload string) string {

	//create empty return string

	// GLL          Geographic position, Latitude and Longitude
	// 4916.46,N    Latitude 49 deg. 16.45 min. North
	// 12311.12,W   Longitude 123 deg. 11.12 min. West

	// //latitude
	// buffer_compress = NMEA_params.rmc_datas.latitude.N_S_indicator;
	// buffer_compress = buffer_compress << 7;
	// buffer_compress += (NMEA_params.rmc_datas.latitude.degrees & 0x7F);
	// NMEA_params.datas_compress_rmc[5] = buffer_compress;

	// NMEA_params.datas_compress_rmc[6] = NMEA_params.rmc_datas.latitude.min_int << 2;

	// buffer_compress = ((NMEA_params.rmc_datas.latitude.min_deci >> 15) & 0x03);
	// NMEA_params.datas_compress_rmc[6] += buffer_compress;

	// NMEA_params.datas_compress_rmc[7] = (NMEA_params.rmc_datas.latitude.min_deci>>7);
	// NMEA_params.datas_compress_rmc[8] = ((NMEA_params.rmc_datas.latitude.min_deci<<1) & 0xFE);

	//B5 : sign + deg
	n, err := strconv.ParseUint(payload[:2], 16, 16) //DEG to bytes

	if err != nil {
		panic(err)
	}

	sign := ""
	if (n & 0x80) != 0 {
		sign = ""
	} else {
		sign = "-"
	}

	//extract coord
	deg := (int)(n & 0x7f)
	degStr := strconv.Itoa(deg)

	//B6 B7 B7  min
	n, err = strconv.ParseUint(payload[2:], 16, 24)

	if err != nil {
		panic(err)
	}

	min_int := (float64)(((n & 0xFC0000) >> 18)) / 60
	min_dec := (float64)((n&0x03FFFE)>>1) / 100000 / 60

	minStr := strconv.FormatFloat(min_int+min_dec, 'f', 5, 64)

	return (sign + degStr + "." + minStr[2:])
}

func parseGpsLong(payload string) string {

	//create empty return string

	//longitude
	// buffer_compress = NMEA_params.rmc_datas.longitude.E_W_indicator;
	// buffer_compress = buffer_compress << 7;
	// buffer_compress += ((NMEA_params.rmc_datas.longitude.degrees >> 1) & 0x7F);
	// NMEA_params.datas_compress_rmc[9] = buffer_compress;

	// buffer_compress = (NMEA_params.rmc_datas.longitude.degrees & 0x01);
	// buffer_compress = buffer_compress << 7;
	// buffer_compress += (NMEA_params.rmc_datas.longitude.min_int << 1);

	// buffer_compress += ((NMEA_params.rmc_datas.longitude.min_deci >>16) & 0x01);
	// NMEA_params.datas_compress_rmc[10] = buffer_compress;

	// NMEA_params.datas_compress_rmc[11] = (NMEA_params.rmc_datas.longitude.min_deci >> 8) & 0xFF;
	// NMEA_params.datas_compress_rmc[12] = (NMEA_params.rmc_datas.longitude.min_deci) & 0xFF;

	// DDMM.MMMMM

	n, err := strconv.ParseUint(payload, 16, 32) //DEG to bytes

	if err != nil {
		panic(err)
	}

	sign := ""
	if (n & 0x80000000) != 0 {
		sign = ""
	} else {
		sign = "-"
	}

	//extract coord
	deg := (int)(n&(0x7f800000)) >> 23
	degStr := strconv.Itoa(deg)

	min_int := (float64)((n&0x007E0000)>>17) / 60
	min_dec := (float64)(n&0x01FFFF) / 100000 / 60

	min := min_int + min_dec

	minStr := strconv.FormatFloat(min, 'f', 5, 64)

	return (sign + degStr + "." + minStr[2:])

}

func parseAutoguidageFrame(payload string) {

	//parse frame

	//check integrity
	if checkFrameIntegrity(payload) != 1 {

		fmt.Println("integrity error : exit")
		return

	}

	frameID := payload[20:22]

	switch frameID {

	case "F6":
		fmt.Println("Rx Status Frame")

		idSrc := payload[6:18]
		influxParam := idSrc + ",beaconId=" + "0" + " " + "frame=\"boot\""
		fmt.Println("Influx Req :", influxParam)
		go putInflux(influxParam)
		go sendSlack("[BOOT][" + idSrc + "]")

		break

	case "01":
		fmt.Println("Rx DATA Frame")

		idSrc := payload[6:18]
		beaconId := payload[24:36]

		fmt.Println("idSrc:", idSrc)
		fmt.Println("beaconId:", beaconId)

		lat := parseGpsLat(payload[36:44])
		long := parseGpsLong(payload[44:52])

		fmt.Println("Lat:", lat)
		fmt.Println("Long:", long)

		//go routine to push in influx here

		influxParam := idSrc + ",beaconId=" + beaconId + " " + "lat=\"" + lat + "\",long=\"" + long + "\""

		fmt.Println("Influx Req :", influxParam)

		go putInflux(influxParam)
		go sendSlack("[GPS][" + idSrc + "] beaconId=" + beaconId + ",lat=" + lat + ",long=" + long)

		break

	}

}

func autoguidage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // parse arguments, you have to call this by yourself
	fmt.Println(time.Now().String())
	fmt.Println(r.Form) // print form information in server side
	fmt.Println("path", r.URL.Path)
	fmt.Println("scheme", r.URL.Scheme)
	fmt.Println(r.Form["url_long"])
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))

		if k == "ntf" {
			parseAutoguidageFrame(strings.Join(v, ""))
		}
	}
	fmt.Fprintf(w, "ok") // send data to client side

}

func main() {

	/*INFLUX*/
	if value, ok := os.LookupEnv("INFLUX_DB_NAME"); ok {
		influxdbConfig.dbname = value
	}

	if value, ok := os.LookupEnv("INFLUX_DB_HOST"); ok {
		influxdbConfig.url = value
	}

	if value, ok := os.LookupEnv("INFLUX_DB_LOGIN"); ok {
		influxdbConfig.login = value
	}

	if value, ok := os.LookupEnv("INFLUX_DB_PWD"); ok {
		influxdbConfig.pwd = value
	}

	if value, ok := os.LookupEnv("SLACK_WEBHOOK_URL"); ok {
		slackWebHookURL = value
	} else {
		slackWebHookURL = SLACKWEBHOOK_URL
	}
	/*print current config*/

	fmt.Printf("INFLUX_DB_NAME : %s\r\n", influxdbConfig.dbname)
	fmt.Printf("INFLUX_DB_LOGIN : %s\r\n", influxdbConfig.login)
	fmt.Printf("INFLUX_DB_PWD: %s\r\n", influxdbConfig.pwd)
	fmt.Printf("INFLUX_DB_URL : %s\r\n", influxdbConfig.url)

	http.HandleFunc("/", autoguidage)        // set router
	err := http.ListenAndServe(":9090", nil) // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
