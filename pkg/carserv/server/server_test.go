package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/mux"
	"github.com/nekruzvatanshoev/carserv/pkg/carserv/dal"
)

func TestServer(t *testing.T) {

	tests := []struct {
		name     string
		path     string
		expected dal.CarResponse
	}{
		{
			name: "MakeOnly",
			path: "/cars?make=Ford",
			expected: dal.CarResponse{

				TotalVehicles:          64201,
				MakeModelTotalVehicles: 64201,
				Suggestions:            []dal.Car{},
			},
		},
		{
			name: "MakeModelOnly",
			path: "/cars?make=Ford&model=Van",
			expected: dal.CarResponse{
				TotalVehicles:          64201,
				MakeModelTotalVehicles: 16,
				Suggestions:            []dal.Car{},
			},
		},

		{
			name: "MakeModelBudgetOnly",
			path: "/cars?make=Ford&model=Van&budget=40000",
			expected: dal.CarResponse{
				TotalVehicles:          147420,
				MakeModelTotalVehicles: 16,
				Lowest:                 36003,
				Median:                 39616,
				Highest:                43665,
				Suggestions: []dal.Car{
					{
						Make:         "Cadillac",
						Model:        "Escalade",
						Price:        36003,
						Year:         2021,
						VehicleCount: 1358,
					},
					{
						Make:         "Audi",
						Model:        "A7",
						Price:        36027,
						Year:         2021,
						VehicleCount: 1358,
					},
					{
						Make:         "Honda",
						Model:        "Civic",
						Price:        36052,
						Year:         2017,
						VehicleCount: 24,
					},
					{
						Make:         "Acura",
						Model:        "TLX",
						Price:        36118,
						Year:         2018,
						VehicleCount: 133,
					},
					{
						Make:         "Subaru",
						Model:        "WRX",
						Price:        36124,
						Year:         2018,
						VehicleCount: 19,
					},
				},
			},
		},
	}

	server := newHTTPServer()
	r := mux.NewRouter()
	r.HandleFunc("/cars", server.GetCars).Methods(http.MethodGet)

	ts := httptest.NewServer(r)

	defer ts.Close()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var carResp dal.CarResponse
			resp, err := http.Get(ts.URL + tc.path)
			if err != nil {
				log.Fatal(err)
			}
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			err = json.Unmarshal(respBody, &carResp)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			if reflect.DeepEqual(carResp, tc.expected) {
				t.Errorf("Expected: %v, Got: %v", tc.expected, carResp)
			}

		})
	}
}
