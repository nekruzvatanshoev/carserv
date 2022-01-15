package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/nekruzvatanshoev/carserv/pkg/carserv/dal"
)

// GetCars defines a GET handler to fetch cars from dataset
func (h *httpServer) GetCars(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	w.Header().Add("Content-Type", "application/json")

	modelName, err := validateModelName(w, vars)
	if err != nil {
		h.log.Printf("model name validation failed: %v", err)
		return
	}

	makeName, err := validateMakeName(w, vars)
	if err != nil {
		h.log.Printf("make name validation failed: %v", err)
		return
	}

	budget, err := validateBudget(w, vars)
	if err != nil {
		h.log.Printf("budget validation failed: %v", err)
		return
	}

	year, err := validateYear(w, vars)
	if err != nil {
		h.log.Printf("year validation failed: %v", err)
		return
	}

	car := dal.Car{
		Make:  makeName,
		Model: modelName,
		Price: budget,
		Year:  year,
	}

	cars := processor(car)

	err = json.NewEncoder(w).Encode(cars)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

}

func validateYear(w http.ResponseWriter, vars url.Values) (int, error) {
	year := vars.Get("year")
	if year != "" {
		yearInt, err := strconv.Atoi(year)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return 0, err
		}
		if yearInt < 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("year must be a positive number: %d", yearInt)))
			return 0, errors.New("year must be a positive number")
		}
		return yearInt, nil
	}
	return 0, nil
}

func validateBudget(w http.ResponseWriter, vars url.Values) (float32, error) {
	budget := vars.Get("budget")
	if budget != "" {
		budgetDecimal, err := strconv.ParseFloat(budget, 32)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return 0.0, err
		}
		if budgetDecimal < 0.0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("budget must be a positive number: %v", budgetDecimal)))
			return 0.0, errors.New("budget must be a positive number")

		}
		return float32(budgetDecimal), nil
	}
	return 0.0, nil
}

func validateMakeName(w http.ResponseWriter, vars url.Values) (string, error) {
	makeName := vars.Get("make")
	if makeName != "" {
		// TODO: Add better validation for non alphanumeric values
		return makeName, nil
	}
	return "", nil
}

func validateModelName(w http.ResponseWriter, vars url.Values) (string, error) {
	modelName := vars.Get("model")
	if modelName != "" {
		// TODO: Add better validation for non alphanumeric values
		return modelName, nil
	}
	return "", nil
}

func processor(car dal.Car) dal.CarResponse {
	// TODO: Add timeouts at the top to timeout if the requests takes too long to process

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
					if makeMatch(i, car.Make) {
						matches <- i
					} else if modelMatch(i, car.Model) {
						matches <- i
					} else if budgetMatch(i, car.Price) {
						matches <- i
					} else if yearMatch(i, car.Year) {
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
					if makeMatch(i, makeName) {
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
					if modelMatch(i, modelName) {
						filterModelName <- i
					}

				}
			}
		}()
		return filterModelName
	}

	filterBudget := func(done <-chan interface{}, intStream <-chan int, budget float32) <-chan int {
		if budget <= 0.0 {
			return intStream
		}

		filterBudgetAmount := make(chan int)
		go func() {
			defer close(filterBudgetAmount)
			for i := range intStream {
				select {
				case <-done:
				default:
					if budgetMatch(i, budget) {
						filterBudgetAmount <- i
					}

				}
			}
		}()
		return filterBudgetAmount
	}

	filterDistinctMake := func(done <-chan interface{}, intStream <-chan int, cars []dal.Car) <-chan int {
		if len(cars) == 0 {
			return intStream
		}

		brands := make(map[string]int)
		filterDistinctMakeStream := make(chan int)
		go func() {
			defer close(filterDistinctMakeStream)
			for i := range intStream {
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

	/* TODO: Possible solution
	 for v := range filterYear(done, filterBudget(done, filterMake(done, filterModel(done, generator(done, dbSize), car.Model), car.Make), car.Price), car.Year) {
		val := dal.CarsDataset[v]
		log.Println(val)
		// rest of the code
	 }
	*/
	_ = func(done <-chan interface{}, intStream <-chan int, year int) <-chan int {
		if year <= 0 {
			return intStream
		}
		filterYear := make(chan int)
		go func() {
			defer close(filterYear)
			for i := range intStream {
				select {
				case <-done:
				default:
					if yearMatch(i, year) {
						filterYear <- i
					}

				}
			}
		}()
		return filterYear
	}

	done := make(chan interface{})
	dbSize := len(dal.CarsDataset)

	var totalVehicles int
	var vehiclePricesCar []dal.Car
	var totalVehiclesMakeModel int

	// Total Number of vehicles available that matches the faceted search parameters (Our OR operations)
	for v := range filterAll(done, generator(done, dbSize), car) {
		val := dal.CarsDataset[v]
		totalVehicles += val.VehicleCount
	}

	// Lowest, Median, and Highest Price of the vehicle that matches the price
	for v := range filterBudget(done, generator(done, dbSize), car.Price) {
		val := dal.CarsDataset[v]
		vehiclePricesCar = append(vehiclePricesCar, val)
	}

	resp := findStatsStruct(vehiclePricesCar)
	resp.TotalVehicles = totalVehicles

	// Number of vehicles matched by Make and Model combination as a sub-group of Total Number
	for v := range filterModel(done, filterMake(done, generator(done, dbSize), car.Make), car.Model) {
		val := dal.CarsDataset[v]
		totalVehiclesMakeModel += val.VehicleCount
	}

	resp.MakeModelTotalVehicles = totalVehiclesMakeModel

	resultSorted := mergeSort(vehiclePricesCar)
	// 5 Suggested vehicles that are within a given budget.
	for num := range take(done, filterDistinctMake(done, generator(done, dbSize), resultSorted), 5) {
		resp.Suggestions = append(resp.Suggestions, resultSorted[num])
	}

	return resp
}

func makeMatch(index int, makeName string) bool {
	return makeName != "" && strings.Contains(dal.CarsDataset[index].Make, makeName)
}

func modelMatch(index int, modelName string) bool {
	return modelName != "" && strings.Contains(dal.CarsDataset[index].Model, modelName)
}

func budgetMatch(index int, budget float32) bool {
	above := budget * 1.10
	below := budget * 0.9
	return budget > 0.0 && dal.CarsDataset[index].Price < above && dal.CarsDataset[index].Price > below
}

func yearMatch(index, year int) bool {
	return year > 0 && dal.CarsDataset[index].Year == year
}

func findStatsStruct(carPrices []dal.Car) dal.CarResponse {
	if len(carPrices) == 0 {
		return dal.CarResponse{Lowest: 0, Median: 0, Highest: 0}
	}
	result := mergeSort(carPrices)
	length := len(carPrices)
	median := length / 2
	return dal.CarResponse{Lowest: result[0].Price, Median: result[median].Price, Highest: result[length-1].Price}
}

func mergeSort(arrCar []dal.Car) []dal.Car {
	if len(arrCar) <= 1 {
		return arrCar
	}

	middle := len(arrCar) / 2
	left := mergeSort(arrCar[:middle])
	right := mergeSort(arrCar[middle:])
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
