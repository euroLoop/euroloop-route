package main

import (
	"encoding/json"
	"fmt"
	"github.com/rs/cors"
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

type InitResponse struct {
	PodWeight       float64 `json:"podweight"`
	MaxAcceleration float64 `json:"maxacceleration"`
	LoadingTime     float64 `json:"loadingtime"`
	ElectricityCost float64 `json:"electricitycost"`
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

	mux := http.NewServeMux()

	mux.HandleFunc("/request", requestHandler)
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/init", initHandler)

	handler := cors.Default().Handler(mux)

	http.ListenAndServe(":"+port, handler)
}

// Called when route website is loaded to populate UI input data fields
func initHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type")
	w.Header().Add("content-type", "application/json")

	initData := ReadFromSheet()
	resp, _ := json.Marshal(initData)
	w.Write(resp)
}

// Called when route is updated
func requestHandler(w http.ResponseWriter, r *http.Request) {

	var data RouteData

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization")
	w.Header().Add("content-type", "application/json")

	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(body))

	_ = json.Unmarshal(body, &data)

	capex := calcCapex(data.Length)
	nrPods := calcNumberOfPods(data.Length, data.Velocity, data.Throughput, data.LoadingTime)
	travelTime := calcTravelTime(data.Length, data.Velocity)

	resp, _ := json.Marshal(Response{nrPods, travelTime, capex, 0, 0})
	w.Write(resp)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	data, _ := json.Marshal(pingResponse{"euroloop-route", "ok"})

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
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
