package main

import (
	"net/http"
	"encoding/json"
	"strings"
	"log"
	"time"
	"fmt"
)

var baseUrl = "http://api.openweathermap.org/data/2.5/weather?APPID=569affbd7c5b588e9832e64ff654aa5c"

func main() {

	mw := multiWeatherProvider{
		openWeatherMap{},
		weatherUnderground{apiKey: "your-key-here"},
	}

	//println("Hola mundo")
	http.HandleFunc("/", hello)
	http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		//tomo la ciudad de la url
		city := strings.SplitN(r.URL.Path, "/", 3)[2]

		temp, err := mw.temperatura(city)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"city": city,
			"temp": temp,
			"took": time.Since(begin).String(),
		})
	})
	//agrego nuevo handler a melli
	http.HandleFunc("/meli/item/", func(w http.ResponseWriter, r *http.Request) {
		item := strings.SplitN(r.URL.Path, "/", 4)[3]
		fmt.Printf("Item id buscado es: %s", item)

		data, err := itemMeli(item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(data)
	})
	http.ListenAndServe(":8081", nil)
}

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hola!"))
}

type itemData struct {
	Id                string  `json:"id"`
	SiteID            string  `json:"site_id"`
	Title             string  `json:"title"`
	Subtitle          string  `json:"subtitle"`
	SellerId          float64 `json:"seller_id"`
	Price             float64 `json:"price"`
	InitialQuantity   int16   `json:"initial_quantity"`
	AvailableQuantity int16   `json:"available_quantity"`
	SoldQuantity      int16   `json:"sold_quantity"`
	/**Pictures struct {
		Id        string `json:"id"`
		Url       string `json:"url"`
		SecureUrl string `json:"secure_url"`
		Size      string `json:"size"`
		MaxSize   string `json:"max_size"`
		Quality   string `json:"quality"`
	} `json:"pictures"`*/
}

func itemMeli(item string) (itemData, error) {
	resp, err := http.Get("https://api.mercadolibre.com/items/" + item)
	if err != nil {
		return itemData{}, err
	}

	defer resp.Body.Close()

	var d itemData
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return itemData{}, err
	}

	return d, nil
}

type openWeatherMap struct{}

func (w openWeatherMap) temperatura(ciudad string) (float64, error) {
	resp, err := http.Get(baseUrl + "&q=" + ciudad)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var d struct {
		Main struct {
			Kelvin float64 `json:"temp"`
		} `json:"main"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}
	log.Printf("openWeatherMap: %s: %.2f", ciudad, d.Main.Kelvin)
	return d.Main.Kelvin, nil
}

type weatherUnderground struct {
	apiKey string
}

type multiWeatherProvider [] weatherProvider

func (w multiWeatherProvider) temperatura(ciudad string) (float64, error) {

	temps := make(chan float64, len(w))
	errs := make(chan error, len(w))

	for _, provider := range w {
		go func(p weatherProvider) {
			k, err := p.temperatura(ciudad)

			if err != nil {
				errs <- err
			}
			temps <- k
		}(provider)
	}

	sum := 0.0
	for i := 0; i < len(w); i++ {
		select {
		case temp := <-temps:
			sum += temp
		case err := <-errs:
			return 0, err
		}
	}
	return sum / float64(len(w)), nil
}

func (w weatherUnderground) temperatura(ciudad string) (float64, error) {
	resp, err := http.Get("http://api.wunderground.com/api/" + w.apiKey + "/conditions/q/" + ciudad + ".json")
	if err != nil {
		return 0, nil
	}

	defer resp.Body.Close()

	var d struct {
		Observation struct {
			Celcius float64 `json:"temp_c"`
		} `json:"current_observation"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, nil
	}

	kelvin := d.Observation.Celcius + 273.15
	log.Printf("weatherUnderground: %s: %.2f", ciudad, kelvin)
	return kelvin, nil
}

func temperatura(ciudad string, providers ...weatherProvider) (float64, error) {
	sum := 0.0

	for _, provider := range providers {
		k, err := provider.temperatura(ciudad)
		if err != nil {
			return 0, nil
		}
		sum += k
	}
	return sum / float64(len(providers)), nil
}

type weatherData struct {
	Name string `json:"name"`
	Main struct {
		Kelvin float64 `json:"temp"`
		Press  float64 `json:"pressure"`
	} `json:"main"`
	Syss struct {
		Type    int     `json:"type"`
		SyssID  int32   `json:"id"`
		Message float64 `json:"message"`
		Pais    string  `json:"country"`
		Sunrise int64   `json:"sunrise"`
		Sunset  int64   `json:"sunset"`
	} `json:"sys"`
}

type weatherProvider interface {
	temperatura(ciudad string) (float64, error)
}
