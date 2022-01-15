package dal

// Car defines a car struct
type Car struct {
	Make         string  `json:"make,omitempty"`
	Model        string  `json:"model,omitempty"`
	Price        float32 `josn:"budget,omitmepty"`
	Year         int     `json:"year,omitempty"`
	VehicleCount int     `json:"-"`
}

// CarResponse defines an HTTP response struct
type CarResponse struct {
	TotalVehicles          int     `json:"total_vehicles,omitempty"`
	MakeModelTotalVehicles int     `json:"make_model_total_vehicles,omitempty"`
	Lowest                 float32 `json:"lowest,omitempty"`
	Median                 float32 `json:"median,omitempty"`
	Highest                float32 `json:"highest,omitempty"`
	Suggestions            []Car   `json:"suggestions,omitempty"`
}
