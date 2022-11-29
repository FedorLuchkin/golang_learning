package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
)

type ValCurs struct {
	Valute []struct {
		CharCode string `xml:"CharCode"`
		Value    string `xml:"Value"`
	} `xml:"Valute"`
}

type ValStruct struct {
	ParamName  string
	ParamValue float64
}

type HTMLParamsStruct struct {
	Status string
	Values []ValStruct
}

type MessageStruct struct {
	Field string
	Value string
}

type HTMLMessageStruct struct {
	Status string
	Values []MessageStruct
}

const APIURL = "http://www.cbr.ru/scripts/XML_daily.asp"
const htmlPath = "static/conventer.html"

var valutes []ValStruct

func internalErrorSent(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
	return
}

func getValutes() error {
	req, err := http.NewRequest(http.MethodGet, APIURL, nil)
	if err != nil {
		panic(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	data := regexp.MustCompile(`<\?.+\?>|<Name>\D+<\/Name>`).ReplaceAllString(string(body), "")
	data = regexp.MustCompile(`,`).ReplaceAllString(data, ".")

	v := ValCurs{}
	err = xml.Unmarshal([]byte(data), &v)
	if err != nil {
		return err
	}
	for _, valute := range v.Valute {
		valuteValue, err := strconv.ParseFloat(valute.Value, 64)
		if err != nil {
			return err
		}
		valutes = append(valutes, ValStruct{valute.CharCode, valuteValue})
	}
	valutes = append(valutes, ValStruct{"RUB", 1})
	sort.Slice(valutes[:], func(i, j int) bool {
		return valutes[i].ParamName < valutes[j].ParamName
	})
	return nil
}

func getValuteValue(valuteName string) float64 {
	var value float64 = -1
	for _, valute := range valutes {
		if valute.ParamName == valuteName {
			value = valute.ParamValue
			break
		}
	}
	return value
}

func main() {

	http.HandleFunc("/", mainPage)
	http.HandleFunc("/rate", ratePage)
	http.HandleFunc("/exchange", exchangePage)

	port := ":9999"
	fmt.Println("Server listening on port:", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe", err)
	}
}

func mainPage(w http.ResponseWriter, r *http.Request) {

	err := getValutes()
	if err != nil {
		internalErrorSent(w)
		return
	}

	tmpl, err := template.ParseFiles(htmlPath)
	if err != nil {
		internalErrorSent(w)
		return
	}

	var htmlParams HTMLParamsStruct
	params := r.URL.Query()

	if len(params) == 0 {
		htmlParams = HTMLParamsStruct{"all", valutes}
		if err := tmpl.Execute(w, htmlParams); err != nil {
			internalErrorSent(w)
			return
		}
	}
}

func ratePage(w http.ResponseWriter, r *http.Request) {

	err := getValutes()
	if err != nil {
		internalErrorSent(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	params := r.URL.Query()

	from := getValuteValue(params.Get("from"))
	to := getValuteValue(params.Get("to"))

	if from == -1 || to == -1 {
		message := make(map[string]string)
		if from == -1 {
			message["from"] = "unknown currency"
		}
		if to == -1 {
			message["to"] = "unknown currency"
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(message)
	} else {
		message := make(map[string]float64)
		message["rate"] = to / from
		json.NewEncoder(w).Encode(message)
	}
	return
}

func exchangePage(w http.ResponseWriter, r *http.Request) {

	err := getValutes()
	if err != nil {
		internalErrorSent(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	params := r.URL.Query()

	from := getValuteValue(params.Get("from"))
	to := getValuteValue(params.Get("to"))

	amount, err := strconv.ParseFloat(params.Get("amount"), 64)
	if err != nil {
		amount = 0
	}

	if from == -1 || to == -1 || (amount < 1) {
		w.Header().Set("Content-Type", "application/json")
		message := make(map[string]string)
		if from == -1 {
			message["from"] = "unknown currency"
		}
		if to == -1 {
			message["to"] = "unknown currency"
		}
		if amount < 1 {
			message["amount"] = "invalid value"
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(message)
	} else {
		message := make(map[string]float64)
		message["amount"] = from / to * amount
		json.NewEncoder(w).Encode(message)
	}
	return
}
