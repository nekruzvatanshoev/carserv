package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nekruzvatanshoev/carserv/pkg/carserv/dal"
)

func NewHTTPServer(addr string) *http.Server {
	server := newHTTPServer()
	r := mux.NewRouter()
	r.HandleFunc("/cars", server.GetCars).Methods(http.MethodGet)
	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}

type httpServer struct {
	log *log.Logger
}

func newHTTPServer() *httpServer {

	return &httpServer{
		log: log.New(os.Stdout, "log", log.LstdFlags),
	}
}

func (h *httpServer) GetCars(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	w.Header().Add("Content-Type", "application/json")

	var car dal.Car
	makeName := vars.Get("make")
	if makeName != "" {
		car.Make = makeName
	}
	modelName := vars.Get("model")
	if modelName != "" {
		car.Model = modelName
	}
	budget := vars.Get("budget")
	if budget != "" {
		budgetDecimal, err := strconv.ParseFloat(budget, 32)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		car.Price = float32(budgetDecimal)
	}
	year := vars.Get("year")
	if year != "" {
		yearInt, err := strconv.Atoi(year)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		car.Year = yearInt
	}

	cars := Process(car)

	err := json.NewEncoder(w).Encode(cars)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

}

func Process(car dal.Car) dal.CarResponse {
	generator := func(done <-chan interface{}, size int) <-chan int {
		intStream := make(chan int)
		go func() {
			defer close(intStream)
			for i := 0; i < size; i++ {
				select {
				case <-done:
					return
				case intStream <- i:
				}
			}
		}()
		return intStream
	}

	filterAll := func(done <-chan interface{}, intStream <-chan int, car dal.Car) <-chan int {
		matches := make(chan int)

		go func() {
			defer close(matches)
			for i := range intStream {
				select {
				case <-done:
				default:
					if car.Make != "" && strings.Contains(dal.CarsDataset[i].Make, car.Make) {
						matches <- i
					} else if car.Model != "" && strings.Contains(dal.CarsDataset[i].Model, car.Model) {
						matches <- i
					} else if car.Price != 0.0 && dal.CarsDataset[i].Price < car.Price*1.10 && dal.CarsDataset[i].Price > car.Price*0.90 {
						matches <- i
					} else if car.Year != 0 && dal.CarsDataset[i].Year == car.Year {
						matches <- i
					}
				}
			}
		}()

		return matches
	}

	filterMake := func(done <-chan interface{}, intStream <-chan int, makeName string) <-chan int {
		if makeName == "" {
			return intStream
		}
		filterMakeStream := make(chan int)
		go func() {
			defer close(filterMakeStream)
			for i := range intStream {
				select {
				case <-done:
				default:
					if strings.Contains(dal.CarsDataset[i].Make, makeName) {
						filterMakeStream <- i
					}
				}
			}
		}()
		return filterMakeStream
	}

	filterModel := func(done <-chan interface{}, intStream <-chan int, modelName string) <-chan int {
		if modelName == "" {
			return intStream
		}
		filterModelName := make(chan int)
		go func() {
			defer close(filterModelName)
			for i := range intStream {
				select {
				case <-done:
				default:
					if strings.Contains(dal.CarsDataset[i].Model, modelName) {
						filterModelName <- i
					}

				}
			}
		}()
		return filterModelName
	}

	filterBudget := func(done <-chan interface{}, intStream <-chan int, budget float32) <-chan int {
		if budget <= 0 {
			return intStream
		}
		filterBudgetAmount := make(chan int)
		go func() {
			defer close(filterBudgetAmount)
			for i := range intStream {
				select {
				case <-done:
				default:
					if dal.CarsDataset[i].Price < budget*1.10 && dal.CarsDataset[i].Price > budget*0.90 {
						filterBudgetAmount <- i
					}

				}
			}
		}()
		return filterBudgetAmount
	}

	filterDistinctMake := func(done <-chan interface{}, size int, cars []dal.Car) <-chan int {
		brands := make(map[string]int)
		filterDistinctMakeStream := make(chan int)
		go func() {
			defer close(filterDistinctMakeStream)
			for i := 0; i < size; i++ {
				select {
				case <-done:
				default:
					if _, ok := brands[cars[i].Make]; !ok {
						brands[cars[i].Make] = i
						filterDistinctMakeStream <- i
					}

				}
			}
		}()
		return filterDistinctMakeStream
	}

	take := func(done <-chan interface{}, intStream <-chan int, num int) <-chan int {
		takeStream := make(chan int)
		go func() {
			defer close(takeStream)
			for i := 0; i < num; i++ {
				select {
				case <-done:
					return
				case takeStream <- <-intStream:
				}
			}
		}()
		return takeStream
	}

	// filterYear := func(done <-chan interface{}, intStream <-chan int, year int) <-chan int {
	// 	if year == 0 {
	// 		return intStream
	// 	}
	// 	filterYear := make(chan int)
	// 	go func() {
	// 		defer close(filterYear)
	// 		for i := range intStream {
	// 			select {
	// 			case <-done:
	// 			default:
	// 				if dal.CarsDataset[i].Year == year {
	// 					filterYear <- i
	// 				}

	// 			}
	// 		}
	// 	}()
	// 	return filterYear
	// }

	done := make(chan interface{})

	dbSize := len(dal.CarsDataset)
	intStream := generator(done, dbSize)

	var totalVehicles int

	pipelines := filterAll(done, intStream, car)
	for v := range pipelines {
		val := dal.CarsDataset[v]

		//cars = append(cars, val)
		totalVehicles += val.VehicleCount

	}

	//var cars []dal.Car

	var vehiclePricesCar []dal.Car
	if car.Price != 0.0 {
		intStream2 := generator(done, dbSize)
		pipelines2 := filterBudget(done, intStream2, car.Price)

		for v := range pipelines2 {
			val := dal.CarsDataset[v]

			vehiclePricesCar = append(vehiclePricesCar, val)
		}

	}

	resp := findStatsStruct(vehiclePricesCar)
	resp.TotalVehicles = totalVehicles

	resultSorted := MergeSort(vehiclePricesCar)

	//takeTop5DistinctCars := take(done, filterDistinctMake(done, len(resultSorted), resultSorted), 5)

	if len(resultSorted) != 0 {
		for num := range take(done, filterDistinctMake(done, len(resultSorted), resultSorted), 5) {
			resp.Suggestions = append(resp.Suggestions, resultSorted[num])
		}
	}

	// pipelines2 := filterYear(done, filterBudget(done, filterModel(done, filterMake(done, intStream, car.Make), car.Model), car.Price), car.Year)

	// var cars []dal.Car
	// var totalVehicles int
	// var vehiclePrices []float32
	// log.Println()
	// for v := range pipelines2 {
	// 	val := dal.CarsDataset[v]
	// 	log.Println(val)
	// 	cars = append(cars, val)
	// 	totalVehicles += val.VehicleCount
	// 	vehiclePrices = append(vehiclePrices, val.Price)
	// }

	// log.Println(totalVehicles)
	// log.Println(vehiclePrices)

	// resp := findStats(vehiclePrices)
	// resp.TotalVehicles = totalVehicles
	// log.Println(MergeSort(vehiclePrices)

	intStream3 := generator(done, dbSize)
	pipelines3 := filterModel(done, filterMake(done, intStream3, car.Make), car.Model)

	var totalVehiclesMakeModel int
	for v := range pipelines3 {
		val := dal.CarsDataset[v]

		// cars = append(cars, val)
		totalVehiclesMakeModel += val.VehicleCount
	}

	resp.MakeModelTotalVehicles = totalVehiclesMakeModel

	var totalCarsCount int
	for _, v := range dal.CarsDataset {
		if car.Make != "" && strings.Contains(v.Make, car.Make) {
			totalCarsCount += v.VehicleCount
		} else if car.Model != "" && strings.Contains(v.Model, car.Model) {
			totalCarsCount += v.VehicleCount
		} else if car.Price != 0.0 && v.Price < car.Price*1.10 && v.Price > car.Price*0.90 {
			totalCarsCount += v.VehicleCount
		} else if car.Year != 0 && v.Year == car.Year {
			totalCarsCount += v.VehicleCount
		}
	}

	return resp
}

func findStatsStruct(carPrices []dal.Car) dal.CarResponse {
	if len(carPrices) == 0 {
		return dal.CarResponse{Lowest: 0, Median: 0, Highest: 0}
	}
	result := MergeSort(carPrices)
	length := len(carPrices)
	median := length / 2
	return dal.CarResponse{Lowest: result[0].Price, Median: result[median].Price, Highest: result[length-1].Price}
}

func MergeSort(arrCar []dal.Car) []dal.Car {
	if len(arrCar) <= 1 {
		return arrCar
	}

	middle := len(arrCar) / 2
	left := MergeSort(arrCar[:middle])
	right := MergeSort(arrCar[middle:])
	return merge(left, right)
}

func merge(left, right []dal.Car) []dal.Car {
	result := make([]dal.Car, len(left)+len(right))
	for i := 0; len(left) > 0 || len(right) > 0; i++ {
		if len(left) > 0 && len(right) > 0 {
			if left[0].Price < right[0].Price {
				result[i] = left[0]
				left = left[1:]
			} else {
				result[i] = right[0]
				right = right[1:]
			}
		} else if len(left) > 0 {
			result[i] = left[0]
			left = left[1:]
		} else if len(right) > 0 {
			result[i] = right[0]
			right = right[1:]
		}
	}
	return result
}
