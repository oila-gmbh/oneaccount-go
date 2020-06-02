package oneaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type contextKey string

const (
	// DataKey key is used to retrieve data sent by the user on authorization from the request context
	DataKey = contextKey("OneAccountData")
)

// Data is used to retrieve data sent by the user on authorization
func Data(r *http.Request) []byte {
	data := r.Context().Value(DataKey)
	b, _ := data.([]byte)
	return b
}

type Setter func(ctx context.Context, k string, v interface{}) error
type Getter func(ctx context.Context, k string) (interface{}, error)

type Engine interface {
	Set(ctx context.Context, k string, v interface{}) error
	Get(ctx context.Context, k string) (interface{}, error)
}

type OneAccount struct {
	Engine             Engine
	GetterSetterEngine *GetterSetterEngine
	CallbackURL        string
}

func New(options ...option) *OneAccount {
	oa := OneAccount{}
	for _, option := range options {
		option(&oa)
	}
	if oa.Engine == nil && oa.GetterSetterEngine == nil {
		oa.Engine = NewInMemoryEngine()
	} else if oa.Engine == nil {
		oa.Engine = oa.GetterSetterEngine
	}

	if oa.CallbackURL == "" {
		oa.CallbackURL = "oneaccountauth"
	}

	return &oa
}

func (oa *OneAccount) save(ctx context.Context, body io.ReadCloser) error {
	var data map[string]interface{}
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		return fmt.Errorf("cannot parse request body")
	}
	if uuid, ok := data["uuid"]; ok {
		uuid, ok := uuid.(string)
		if !ok || uuid == "" {
			return fmt.Errorf("incorrect uuid")
		}
		delete(data, "uuid")
		delete(data, "externalId")
		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("error marshalling data from body: %v", err)
		}
		err = oa.Engine.Set(ctx, uuid, b)
		if err != nil {
			return fmt.Errorf("engine error: cannot set")
		}
		return nil
	}
	return fmt.Errorf("uuid is required")
}

func (oa *OneAccount) authorize(ctx context.Context, r *http.Request, token, uuid string) (interface{}, error) {
	if token == "" {
		return nil, fmt.Errorf("empty or wrong bearer token")
	}
	if uuid == "" {
		return nil, fmt.Errorf("uuid is not provided")
	}

	v, err := oa.Engine.Get(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("engine error: key is not found")
	}
	// TODO: verify the token
	return v, nil
}

// Auth handles the authentication
func (oa *OneAccount) Auth(next http.Handler) http.Handler {
	hfc := func(w http.ResponseWriter, r *http.Request) {
		if oa == nil || r.URL.Path != oa.CallbackURL || oa.Engine == nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		token, err := BearerFromHeader(r)
		if err != nil || token == "" {
			err := oa.save(ctx, r.Body)
			if err != nil {
				Error(w, err, http.StatusBadRequest)
				return
			}
			JSON(w, "{success: true}")
			return
		}
		data, err := oa.authorize(ctx, r, token, r.Header.Get("uuid"))
		if err != nil {
			Error(w, err, http.StatusBadRequest)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), DataKey, data))
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(hfc)
}

func IsAuthenticated(r *http.Request) bool {
	return Data(r) != nil
}
