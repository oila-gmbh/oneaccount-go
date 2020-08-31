package oneaccount

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// BearerFromHeader method
// retrieves token from BEARER header
func BearerFromHeader(r *http.Request) (string, error) {

	auth := r.Header.Get("Authorization")
	const prefix = "BEARER "

	if !(len(auth) >= len(prefix) && strings.ToUpper(auth[0:len(prefix)]) == prefix) {
		return "", errors.New("token is required")
	}

	t := auth[len(prefix):]

	return t, nil
}

// JSON .
func JSON(w http.ResponseWriter, v interface{}, status ...int) {
	if len(status) > 0 {
		w.WriteHeader(status[0])
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		// TODO: log this
		Error(w, fmt.Errorf("cannot send response"), http.StatusInternalServerError)
	}
}

// Error .
func Error(w http.ResponseWriter, err error, status ...int) {
	if err == nil {
		err = errors.New("unknown error")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if len(status) > 0 {
		w.WriteHeader(status[0])
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	fmt.Fprintf(w, `{"error":%q}`, err)
}
