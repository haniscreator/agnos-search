package main

import (
	"encoding/json"
	"net/http"
)

func main() {
	http.HandleFunc("/patient/search/", func(w http.ResponseWriter, r *http.Request) {
		// extract id from URL
		id := r.URL.Path[len("/patient/search/"):]

		// Return fake hospital patient response
		resp := map[string]any{
			"first_name_th":  "มานพ",
			"middle_name_th": "",
			"last_name_th":   "สุขใจ",
			"first_name_en":  "Manop",
			"middle_name_en": "",
			"last_name_en":   "Sukjai",
			"date_of_birth":  "1985-05-05",
			"patient_hn":     "HN-999",
			"national_id":    id,
			"passport_id":    "",
			"phone_number":   "0811112222",
			"email":          "manop@example.com",
			"gender":         "M",
		}

		json.NewEncoder(w).Encode(resp)
	})

	println("Mock hospital API running on http://localhost:8081")
	http.ListenAndServe(":8081", nil)
}
