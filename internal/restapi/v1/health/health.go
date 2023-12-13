package v1health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"

	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/restapi/media"
)

func Create(server chi.Router) {
	server.Get("/info/health", healthGet)
	server.Get("/", healthGet)
}

func healthGet(w http.ResponseWriter, r *http.Request) {
	dto := HealthResultDto{Status: "up"}

	w.Header().Add(headers.ContentType, media.ContentTypeApplicationJson)
	w.WriteHeader(http.StatusOK)
	writeJson(r.Context(), w, dto)
}

func writeJson(ctx context.Context, w http.ResponseWriter, v interface{}) {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)
	if err != nil {
		logging.LoggerFromContext(ctx).Warn(fmt.Sprintf("error while encoding json response: %v", err))
	}
}
