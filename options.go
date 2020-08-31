package oneaccount

import (
	"context"
	"fmt"
	"net/http"
)

type option func(oa *OneAccount)

func SetOnErrorListener(errorListener ErrorListener) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.ErrorListener = errorListener
	}
}

func SetEngine(e Engine) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.Engine = e
	}
}

func SetCallbackURL(callbackURL string) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.CallbackURL = &callbackURL
	}
}

func SetClient(client *http.Client) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.Client = client
	}
}

type GetterSetterEngine struct {
	Setter Setter
	Getter Getter
}

func (g GetterSetterEngine) Set(ctx context.Context, k string, v []byte) error {
	if g.Setter == nil {
		return fmt.Errorf("engine setter is not set")
	}
	return g.Setter(ctx, k, v)
}

func (g GetterSetterEngine) Get(ctx context.Context, k string) ([]byte, error) {
	if g.Getter == nil {
		return nil, fmt.Errorf("engine getter is not set")
	}
	return g.Getter(ctx, k)
}

func SetEngineSetter(setter Setter) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.Engine = nil
		if oa.GetterSetterEngine == nil {
			oa.GetterSetterEngine = &GetterSetterEngine{}
		}
		oa.GetterSetterEngine.Setter = setter
	}
}

func SetEngineGetter(getter Getter) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.Engine = nil
		if oa.GetterSetterEngine == nil {
			oa.GetterSetterEngine = &GetterSetterEngine{}
		}
		oa.GetterSetterEngine.Getter = getter
	}
}
