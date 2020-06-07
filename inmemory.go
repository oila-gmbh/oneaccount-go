package oneaccount

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const ExpireTimeDuration = 1 * time.Minute

type AuthorizingUser struct {
	ExpiresAt time.Time
	Data      []byte
}

type InMemoryEngine struct {
	AuthorizingUsers map[string]AuthorizingUser
	sync.RWMutex
}

func NewInMemoryEngine() *InMemoryEngine {
	i := &InMemoryEngine{}
	i.AuthorizingUsers = make(map[string]AuthorizingUser)
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				i.Lock()
				now := time.Now()
				for k, v := range i.AuthorizingUsers {
					if v.ExpiresAt.Before(now) {
						delete(i.AuthorizingUsers, k)
					}
				}
				i.Unlock()
			}
		}
	}()
	return i
}

func (i *InMemoryEngine) Set(ctx context.Context, k string, v []byte) error {
	// we don't need a sophisticated way to handle context here, so we just check
	// if it is already cancelled and return if it is otherwise ignore the context and proceed
	select {
	case <-ctx.Done():
		return nil
	default:
		i.Lock()
		i.AuthorizingUsers[k] = AuthorizingUser{
			ExpiresAt: time.Now().Add(ExpireTimeDuration),
			Data:      v,
		}
		i.Unlock()
	}

	return nil
}

func (i *InMemoryEngine) Get(ctx context.Context, k string) ([]byte, error) {
	// we don't need a sophisticated way to handle context here, so we just check
	// if it is already cancelled and return if it is otherwise ignore the context and proceed
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		i.RLock()
		v, ok := i.AuthorizingUsers[k]
		i.RUnlock()
		if !ok || v.ExpiresAt.Before(time.Now()){
			return nil, fmt.Errorf("no item found or item expired for key: %s", k)
		}
		defer func() {
			i.Lock()
			delete(i.AuthorizingUsers, k)
			i.Unlock()
		}()
		return v.Data, nil
	}
}
