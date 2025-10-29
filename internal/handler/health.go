package handler

import (
	"context"
	"net/http"
	"reflect"
	"time"
)

// DBPinger is a minimal DB connectivity interface.
type DBPinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db DBPinger
}

func NewHealthHandler(db DBPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// PingHandler checks DB connectivity and returns 200 OK on success, 500 otherwise.
func (h *HealthHandler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if h.db == nil || isTypedNil(h.db) {
		writePlain(w, http.StatusInternalServerError, "DB not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		writePlain(w, http.StatusInternalServerError, "DB ping failed")
		return
	}
	writePlain(w, http.StatusOK, "OK")
}

func isTypedNil(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
		return v.IsNil()
	default:
		return false
	}
}
