package main

import (
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

	var err error
	err = getValutes()
	if err != nil {
		log.Fatal("getValutes", err)
	}

	port := ":8888"
	fmt.Println("Server listening on port:", port)
	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe", err)
	}
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(htmlPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Fatal("ParseHTMLFiles", err)
		return
	}

	var htmlParams HTMLParamsStruct
	params := r.URL.Query()

	if len(params) == 0 {
		htmlParams = HTMLParamsStruct{"all", valutes}
		if err := tmpl.Execute(w, htmlParams); err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatal("templateExecuting", err)
			return
		}
	}
}

func ratePage(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	tmpl, err := template.ParseFiles(htmlPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Fatal("ParseHTMLFiles", err)
		return
	}

	from := getValuteValue(params.Get("from"))
	to := getValuteValue(params.Get("to"))

	if from == -1 || to == -1 {
		var htmlErrors []MessageStruct
		if from == -1 {
			htmlErrors = append(htmlErrors, MessageStruct{"from", "unknown currency"})
		}
		if to == -1 {
			htmlErrors = append(htmlErrors, MessageStruct{"to", "unknown currency"})
		}
		if err := tmpl.Execute(w, HTMLMessageStruct{"error", htmlErrors}); err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatal("templateExecuting", err)
			return
		}
	} else {
		messages := HTMLMessageStruct{"rate",
			[]MessageStruct{
				{params.Get("from") + "  -->", params.Get("to")},
				{"amount", string(fmt.Sprintf("%.3f", to/from))},
			}}
		if err := tmpl.Execute(w, messages); err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatal("templateExecuting", err)
			return
		}
	}

}

func exchangePage(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	tmpl, err := template.ParseFiles(htmlPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Fatal("ParseHTMLFiles", err)
		return
	}

	from := getValuteValue(params.Get("from"))
	to := getValuteValue(params.Get("to"))
	amount, err := strconv.ParseFloat(params.Get("amount"), 64)
	if err != nil {
		amount = 0
	}

	if from == -1 || to == -1 || (amount < 1) {
		var htmlErrors []MessageStruct
		if from == -1 {
			htmlErrors = append(htmlErrors, MessageStruct{"from", "unknown currency"})
		}
		if to == -1 {
			htmlErrors = append(htmlErrors, MessageStruct{"to", "unknown currency"})
		}
		if amount < 1 {
			htmlErrors = append(htmlErrors, MessageStruct{"amount", "invalid value"})
		}
		if err := tmpl.Execute(w, HTMLMessageStruct{"error", htmlErrors}); err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatal("templateExecuting", err)
			return
		}
	} else {
		messages := HTMLMessageStruct{"amount",
			[]MessageStruct{
				{params.Get("from") + "  -->", params.Get("to")},
				{"amount", params.Get("amount")},
				{"result", string(fmt.Sprintf("%.3f", from/to*amount))},
			}}
		if err := tmpl.Execute(w, messages); err != nil {
			http.Error(w, err.Error(), 500)
			log.Fatal("templateExecuting", err)
			return
		}
	}
}
