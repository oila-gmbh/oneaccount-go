### oneaccount-go

This library is a middleware for the golang http router (or any HTTP request multiplexer that follow standard library semantics).

Please follow the instructions or official documentations for an integration.

#### NOTE: examples 2 and 3 are the most preferred approach for a production setup

#### Example 1 (In Memory Engine):
`oneaccount-go` by default uses in memory cache engine if a custom engine is not supplied.
```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/Kiura/oneaccount-go"
    "os"
)

// The callback URL is the URL you have set when you created One account app.
// The pattern for the router, callbackURL passed to SetCallbackURL function
// and callback URL of the application must be the same.
func main() {
    http.Handle("/oneaccountauth", oneaccount.New(
        oneaccount.SetCallbackURL("/oneaccountauth"),
    ).Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
        userID, _ := data["userId"]
        // the same way you can access any other data you requested from the user:
        firstName, _ := data["firstName"]
        // or create a struct to extract the data to
	// any data returned here would be sent to onAuth function on front-end e.g.:
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{"firstName": firstName)
    })))
}
```

For brevity we will leave out comments for the following examples, 
if something is unclear please read the comments on the first example 
or check the documentation or create an issue 

Next 2 examples show how you can use any caching engine with `oneaccount-go`
this is recommended for a production environment. Both examples are used
for the same purpose the only difference is how you implement them.
We will be using redis for these examples: https://github.com/go-redis/redis

#### Example 2 (Custom Engine functions):
```go
func main() {
    http.Handle("/oneaccountauth", oneaccount.New(
        oneaccount.SetCallbackURL("/oneaccountauth"),
        oneaccount.SetEngineSetter(func(ctx context.Context, k string, v interface{}) error {
            b, err := json.Marshal(v)
            if err != nil {
                return err
            }
            return redisClient.Set(ctx, k, b, 60 * time.Second).Err()
        }),
        oneaccount.SetEngineGetter(func(ctx context.Context, k string) (interface{}, error) {
	    v, err := ore.client.Get(ctx, k).Result()
	    if err != nil {
		return nil, err
	    }
	    return v, ore.client.Del(ctx, k).Err()
        }),
    ).Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func (ore OneaccountRedisEngine) Set(ctx context.Context, k string, v interface{}) error {
    b, err := json.Marshal(v)
    if err != nil {
        return err
    }
    return ore.client.Set(ctx, k, b, 60 * time.Second).Err()
}

func (ore OneaccountRedisEngine) Get(ctx context.Context, k string) (interface{}, error) {
    v, err := ore.client.Get(ctx, k).Result()
    if err != nil {
        return nil, err
    }
    return v, ore.client.Del(ctx, k).Err()
}

func NewOneaccountRedisEngine(redisClient *redis.Client) *OneaccountRedisEngine {
    return &OneaccountRedisEngine{
	    client: redisClient,
    }
}

func main() {
    var redisClient = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    
    http.Handle("/oneaccountauth", oneaccount.New(
        oneaccount.SetCallbackURL("/oneaccountauth"),
        oneaccount.SetEngine(NewOneaccountRedisEngine(redisClient),
    ).Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !oneaccount.IsAuthenticated(r) {
            return
        }
    })))
}
```

This example is longer, but it allows a greater control 
and is easier to separate the logic into a separate file
