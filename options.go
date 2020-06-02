package oneaccount

import (
	"context"
	"fmt"
)

type option func(oa *OneAccount)

func SetEngine(e Engine) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.Engine = e
	}
}

func SetCallbackURL(callbackURL string) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.CallbackURL = callbackURL
	}
}

type GetterSetterEngine struct {
	Setter Setter
	Getter Getter
}

func (g GetterSetterEngine) Set(ctx context.Context, k string, v interface{}) error {
	if g.Setter == nil {
		return fmt.Errorf("setter is not set")
	}
	return g.Setter(ctx, k, v)
}

func (g GetterSetterEngine) Get(ctx context.Context, k string) (interface{}, error) {
	if g.Getter == nil {
		return nil, fmt.Errorf("getter is not set")
	}
	return g.Getter(ctx, k)
}

func SetEngineSetter(setter Setter) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.Engine = nil
		oa.GetterSetterEngine = &GetterSetterEngine{}
		if oa.GetterSetterEngine == nil {
			oa.GetterSetterEngine = &GetterSetterEngine{}
		}
		oa.GetterSetterEngine.Setter = setter
	}
}

func SetEngineGetter(getter Getter) func(oa *OneAccount) {
	return func(oa *OneAccount) {
		oa.Engine = nil
		oa.GetterSetterEngine = &GetterSetterEngine{}
		if oa.GetterSetterEngine == nil {
			oa.GetterSetterEngine = &GetterSetterEngine{}
		}
		oa.GetterSetterEngine.Getter = getter
	}
}
