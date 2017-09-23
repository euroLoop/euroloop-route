package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
)

type RouteData struct {
	Length      float64 `json:"length"`
	Velocity    float64 `json:"velocity"`
	Throughput  float64 `json:"throughput"`
	Diameter    float64 `json:"diameter"`
	LoadingTime float64 `json:"loadingtime"`
}

type Response struct {
	Nrpods           int `json:"nrpods"`
	TravelTime       int `json:"traveltime"`
	Capex            int `json:"capex"`
	Opex             int `json:"opex"`
	PowerConsumption int `json:"powerconsumption"`
}

type pingResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

var tubeSegmentCost float64 = 28300.0
var tubeJointCost float64 = 8700.0
var tubeSegmentLength float64 = 12.0
var pylonCost float64 = 16800.0
var pylonSpacingM float64 = 20.0

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/request", requestHandler)
	http.HandleFunc("/ping", pingHandler)
	http.ListenAndServe(":"+port, nil)

}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	data, _ := json.Marshal(pingResponse{"euroloop-route", "ok"})

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func requestHandler(w http.ResponseWriter, r *http.Request) {

	var data RouteData
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("access-control-allow-headers", "access-control-allow-headers, origin, content-type, accept")
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(body))
	_ = json.Unmarshal(body, &data)

	capex := calcCapex(data.Length)
	nrPods := calcNumberOfPods(data.Length, data.Velocity, data.Throughput, data.LoadingTime)
	travelTime := calcTravelTime(data.Length, data.Velocity)

	resp, _ := json.Marshal(Response{nrPods, travelTime, capex, 0, 0})
	w.Write(resp)
}

func calcTravelTime(length float64, velocity float64) int {
	return int(math.Ceil((length / 1000) / (velocity / 60)))
}

func calcNumberOfPods(length float64, velocity float64, throughput float64, loadingtime float64) int {

	RTT := (2*length/1000)/(velocity/60) + 2*loadingtime
	containerPerMinute := throughput / (24 * 60)

	return int(math.Ceil(RTT * containerPerMinute))
}

func calcCapex(length float64) int {

	tubeSegments := math.Ceil(length/tubeSegmentLength) * 2
	tubeCost := tubeSegments * (tubeSegmentCost + tubeJointCost)
	pylonCostTotal := pylonCost * math.Ceil(length/pylonSpacingM)

	return int(tubeCost + pylonCostTotal)
}
