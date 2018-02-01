package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/slack"
)

type Route struct {
	Name     string    `json:"name"`
	Segments []segment `json:"coords"`
}

type RouteList struct {
}

type segment struct {
	lat float64 `json:"lat"`
	lng float64 `json:"lng"`
	rad float64 `json:"rad"`
}

type RouteData struct {
	Length      float64 `json:"length"`
	Velocity    float64 `json:"velocity"`
	TravelTime  float64 `json:"travel_time"`
	Throughput  float64 `json:"throughput"`
	Diameter    float64 `json:"diameter"`
	LoadingTime float64 `json:"loadingtime"`
}

type Response struct {
	Nrpods           int `json:"nrpods"`
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

var db *sql.DB

func main() {

	var err error

	connStr := "postgres://qtkplvmtlnoxmo:e6be41ebe5299e829f9be8b59dbe437bb4599721662d40cda2e047117699321f@ec2-23-21-246-25.compute-1.amazonaws.com:5432/damrsuh0tjqpqi"

	db, err = sql.Open("postgres", connStr)
	checkErr(err)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", mainHandler)
	mux.HandleFunc("/request", requestHandler)
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/saveroute", saveRoute)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	handler := cors.Default().Handler(mux)

	http.ListenAndServe(":"+port, handler)

	getRouteNames()
}

func getRouteNames() {

	var id int
	var name string

	rows, err := db.Query("SELECT id, doc->'name' FROM routes")
	defer rows.Close()

	for rows.Next() {

		if err := rows.Scan(&id, &name); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d: %s\n", id, name)
	}
	checkErr(err)
}

func saveRoute(w http.ResponseWriter, r *http.Request) {

	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(body))

	_, err := db.Exec("INSERT INTO routes (doc) VALUES ($1)", string(body))
	checkErr(err)
}

func loadRoutes(w http.ResponseWriter, r *http.Request) {

}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("static/index.html")
	if err != nil {
		log.Print("template parsing error: ", err)
	}
	err = t.Execute(w, "")
	if err != nil {
		log.Print("template executing error: ", err)
	}
}

// Called when route is updated
func requestHandler(w http.ResponseWriter, r *http.Request) {

	var data RouteData

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization")

	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(body))

	_ = json.Unmarshal(body, &data)

	capex := calcCapex(data.Length)
	nrPods := calcNumberOfPods(data.TravelTime, data.Throughput, data.LoadingTime)

	resp, _ := json.Marshal(Response{nrPods, capex, 0, 0})
	w.Write(resp)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	userCode := r.URL.Query().Get("code")
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     "137857628480.302386028709",
		ClientSecret: os.Getenv("SLACK_TOKEN"),
		Scopes:       []string{"identity.basic", "identity.team", "identity.avatar"},
		Endpoint:     slack.Endpoint,
	}

	tok, err := conf.Exchange(ctx, userCode)
	if err != nil {
		fmt.Println("ERROR!!!")
		log.Panic(err)
	} else {
		fmt.Println("Performed OAuth")
		fmt.Println("Token:")
		// fmt.Println(tok)
		// fmt.Println(tok.AccessToken)
		fmt.Println("Team:")
		fmt.Println(tok.Extra("team_id"))
		fmt.Println(tok.Extra("team"))
		fmt.Println("End of login")
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	data, _ := json.Marshal(pingResponse{"euroloop-route", "ok"})

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func calcNumberOfPods(traveltime float64, throughput float64, loadingtime float64) int {

	RTT := (2 * traveltime / 60) + 2*loadingtime
	containerPerMinute := throughput / (24 * 60)

	return int(math.Ceil(RTT * containerPerMinute))
}

func calcCapex(length float64) int {

	tubeSegments := math.Ceil(length/tubeSegmentLength) * 2
	tubeCost := tubeSegments * (tubeSegmentCost + tubeJointCost)
	pylonCostTotal := pylonCost * math.Ceil(length/pylonSpacingM)

	return int(tubeCost + pylonCostTotal)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
