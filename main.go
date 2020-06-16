package main // import github.com/HuiOnePos/flysnow

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/HuiOnePos/flysnow/fly"
	"github.com/HuiOnePos/flysnow/tmp"
	"github.com/HuiOnePos/flysnow/utils"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	utils.LoacConfig()
	tmp.Init()
	go func() {
		logrus.Println(http.ListenAndServe(":7777", nil))
	}()
	fly.StartServer()

}
