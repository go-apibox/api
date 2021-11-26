package main

import (
	"github.com/apibox/api"
	"log"
	"os"
)

func main() {
	cfg := `
# config of application
app:
  name: myapi
  http_addr: :8080
`
	app, err := api.NewAppFromYaml(cfg)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
	app.Run([]*api.Route{
		api.NewRoute("Test.Ok", testOkAction),
		api.NewRoute("Test.Error", testErrorAction),
	})
}

func testErrorAction(c *api.Context) (data interface{}) {
	return c.Error.New(api.ErrorInvalidParam, "Password", "TooShort")
}

func testOkAction(c *api.Context) (data interface{}) {
	return "ok"
}
