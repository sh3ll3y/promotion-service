package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/sh3ll3y/promotion-service/internal/logging"
	"github.com/sh3ll3y/promotion-service/internal/service"
	"go.uber.org/zap"
	"net/http"
)

func RegisterHandlers(router *mux.Router, service *service.PromotionService) {
	router.HandleFunc("/promotions/{id}", getPromotionHandler(service)).Methods("GET")
	router.HandleFunc("/process-csv", processCSVHandler(service)).Methods("POST")
}

func getPromotionHandler(service *service.PromotionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		promotion, err := service.GetPromotion(id)
		if err != nil {
			logging.Logger.Error("Failed to get promotion", zap.Error(err), zap.String("id", id))
			http.Error(w, "Promotion not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(promotion)
	}
}

func processCSVHandler(service *service.PromotionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := r.FormValue("filename")
		if filename == "" {
			http.Error(w, "Filename is required", http.StatusBadRequest)
			return
		}

		err := service.ProcessCSVFile(filename)
		if err != nil {
			logging.Logger.Error("Failed to process CSV file", zap.Error(err), zap.String("filename", filename))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "CSV processed successfully"})
	}
}