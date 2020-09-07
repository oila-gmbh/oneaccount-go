### oneaccount-go

This library is a middleware for the golang http router (or any HTTP request multiplexer that follow standard library semantics).

Please follow the instructions or official documentations for an integration.

#### NOTE: examples 2 and 3 are the most preferred approaches for a production setup. Example 1 is only for development, or a service with small traffic.


#### Example 1 (In Memory Engine):
`oneaccount-go` by default uses in memory cache engine if a custom engine is not supplied.
```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/oilastudio/oneaccount-go"
)

// The route URL is the callback URL you have set when you created One account app.
func main() {
    var oa = oneaccount.New()
    http.Handle("/oneaccountauth", oa.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !oneaccount.IsAuthenticated(r) {
            return
        }
        // user authenticated and you can implement any logic your application 
        // needs. As an example you can extract data sent by the user 
        // after successful authentication
        data := make(map[string]interface{})
        if err := json.Unmarshal(oneaccount.Data(r), &data); err != nil {
            // handle the error
        }
        // since One account doesn't differentiate between sign up and sign in, 
        // you can use userId to check if the user signed up on your website or not
        // userID, _ := data["userId"]
        // the same way you can access any other data you requested from the user:
        firstName, _ := data["firstName"]
        // or create a struct to extract the data to
        // any data returned here would be sent to onAuth function on front-end e.g.:
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        if err := json.NewEncoder(w).Encode(map[string]interface{}{"firstName": firstName}); err != nil {
            // handle the error
        }
    })))
}
```

For brevity, we will leave out comments for the following examples, 
if something is unclear please read the comments on the first example 
or check the documentation or create an issue 

Next 2 examples show how you can use any caching engine with `oneaccount-go`.
This approach is recommended for a production environment. Both examples are used
for the same purpose the only difference is how you implement them.
We will be using redis for these examples: https://github.com/go-redis/redis

#### Example 2 (Custom Engine functions):
```go
func main() {
	var redisClient = redis.NewClient(&redis.Options{})
	var oa = oneaccount.New(
		oneaccount.SetEngineSetter(func(ctx context.Context, k string, v []byte) error {
			// for best results the timeout should match the timeout 
			// set in frontend (updateInterval option, default: 3 minutes)
			return redisClient.Set(ctx, k, v, 3 * time.Minute).Err()
		}),
		oneaccount.SetEngineGetter(func(ctx context.Context, k string) ([]byte, error) {
			v, err := redisClient.Get(ctx, k).Result()
			if err != nil {
				return nil, err
			}
			return []byte(v), redisClient.Del(ctx, k).Err()
		}),
	)
	http.Handle("/oneaccountauth", oa.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !oneaccount.IsAuthenticated(r) {
			return
		}
	})))
}
```
Now our authentication is production ready!

#### Example 3 (Custom Engine):
```go
type OneaccountRedisEngine struct {
    client *redis.Client
}

func (ore OneaccountRedisEngine) Set(ctx context.Context, k string, v []byte) error {
    return ore.client.Set(ctx, k, v, 3 * time.Minute).Err()
}

func (ore OneaccountRedisEngine) Get(ctx context.Context, k string) ([]byte, error) {
    v, err := ore.client.Get(ctx, k).Result()
    if err != nil {
        return nil, err
    }
    return []byte(v), ore.client.Del(ctx, k).Err()
}

func main() {
    var redisClient = redis.NewClient(&redis.Options{})
    var oa = oneaccount.New(
        oneaccount.SetEngine(&OneaccountRedisEngine{client: redisClient}),
    )
    http.Handle("/oneaccountauth", oa.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !oneaccount.IsAuthenticated(r) {
            return
        }
    })))
}
```

This example is a little longer, but it allows a greater control 
and is easier to separate the logic into a separate file.
