package oneaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

type contextKey string

// TODO: improve status code messages
// TODO: improve reported errors to the user

const (
	verifyURL = "https://api.oneaccount.app/widget/verify"
	// DataKey key is used to retrieve data sent by the user on authorization from the request context
	DataKey = contextKey("OneAccountData")
)

// Data is used to retrieve data sent by the user on authorization
func Data(r *http.Request) []byte {
	data := r.Context().Value(DataKey)
	b, _ := data.([]byte)
	return b
}

type Setter func(ctx context.Context, k string, v []byte) error
type Getter func(ctx context.Context, k string) ([]byte, error)
type ErrorListener func(ctx context.Context, err error)

type Engine interface {
	Set(ctx context.Context, k string, v []byte) error
	Get(ctx context.Context, k string) ([]byte, error)
}

type OneAccount struct {
	Engine             Engine
	GetterSetterEngine *GetterSetterEngine
	Client             *http.Client
	CallbackURL        *string
	ErrorListener      ErrorListener
}

func httpClient() *http.Client {
	var netTransport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 15 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	var netClient = &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}
	return netClient
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

	if oa.Client == nil {
		oa.Client = httpClient()
	}

	return &oa
}

func (oa *OneAccount) verify(ctx context.Context, token, uuid string) (err error) {
	var res *http.Response
	var req *http.Request
	data := fmt.Sprintf(`{"uuid": "%s"})`, uuid)
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "BEARER "+token)
	res, err = oa.Client.Do(req)
	if err != nil {
		return
	}
	if res.Body == nil {
		err = fmt.Errorf("request body is empty")
		return
	}
	defer func() {
		cerr := res.Body.Close()
		if err == nil {
			err = cerr
		}
	}()
	if res == nil || res.StatusCode != http.StatusOK {
		err = fmt.Errorf("cannot verify the request")
		return
	}
	type response struct {
		Success bool `json:"success"`
	}
	var resp = response{}
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil || !resp.Success {
		err = fmt.Errorf("cannot verify the request")
	}
	return
}

func (oa *OneAccount) save(ctx context.Context, body []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		return fmt.Errorf("cannot parse request body: %v", err)
	}
	uuidI, ok := data["uuid"]
	if !ok {
		return fmt.Errorf("uuid is required")
	}
	uuid, ok := uuidI.(string)
	if !ok || uuid == "" {
		return fmt.Errorf("incorrect uuid: %s", uuid)
	}
	delete(data, "uuid")
	delete(data, "externalId")
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling data from body, err:%v, data: %v", err, data)
	}
	err = oa.Engine.Set(ctx, uuid, b)
	if err != nil {
		return fmt.Errorf("engine error: cannot set. err: %v, key: %s, value: %s", err, uuid, string(b))
	}
	return nil
}

func (oa *OneAccount) authorize(ctx context.Context, token, uuid string) (interface{}, error) {
	if token == "" {
		return nil, fmt.Errorf("empty or wrong bearer token")
	}
	if uuid == "" {
		return nil, fmt.Errorf("uuid is not provided")
	}

	v, err := oa.Engine.Get(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("engine error: key is not found, err: %v, key: %s, value:%s", err, uuid, string(v))
	}
	if err := oa.verify(ctx, token, uuid); err != nil {
		return nil, err
	}
	return v, nil
}

func (oa *OneAccount) error(ctx context.Context, err error) {
	if err != nil && oa.ErrorListener != nil {
		oa.ErrorListener(ctx, err)
	}
}

// Auth handles the authentication
func (oa *OneAccount) Auth(next http.Handler) http.Handler {
	hfc := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if oa == nil || oa.Engine == nil {
			oa.error(ctx, fmt.Errorf("engine is not provided or the library is not initialised"))
			next.ServeHTTP(w, r)
			return
		}
		if oa.CallbackURL != nil && r.URL.Path != *oa.CallbackURL {
			next.ServeHTTP(w, r)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
        if err != nil {
            oa.error(ctx, fmt.Errorf("cannot decode body,  body: %s, err:%v", string(body), err))
            Error(w, fmt.Errorf("incorrect data is sent"), http.StatusBadRequest)
            return
        }

		token, err := BearerFromHeader(r)
		if err != nil || token == "" {
			err := oa.save(ctx, body)
			if err != nil {
				oa.error(ctx, err)
				Error(w, fmt.Errorf("incorrect data is sent"), http.StatusBadRequest)
				return
			}
			JSON(w, "{success: true}")
			return
		}
		type requestBody struct {
			UUID string `json:"uuid,omitempty"`
		}
		rBody := requestBody{}
		err = json.Unmarshal(body, &rBody)
		if err != nil {
			oa.error(ctx, fmt.Errorf("cannot decode body, body: %s, err: %v", string(body), err))
			Error(w, fmt.Errorf("incorrect data is sent"), http.StatusBadRequest)
			return
		}

		data, err := oa.authorize(ctx, token, rBody.UUID)
		if err != nil {
			oa.error(ctx, fmt.Errorf("cannot authorise: %v", err))
			Error(w, fmt.Errorf("cannot authorise"), http.StatusBadRequest)
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
