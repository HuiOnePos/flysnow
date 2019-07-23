package main

import (
	"fmt"
	"github.com/panjjo/flysnow/utils"
	"github.com/panjjo/go-flysnow"
	"sync"
	"time"
)

func main() {

	// query()
	send()
}
func query() {
	conn, err := flysnow.NewConnection("192.168.1.90", 22258)
	fmt.Println(err)
	res, err := conn.Stat("type", &flysnow.StatQuery{
		Term: "type",
		Index: map[string]interface{}{
			"shopid": "bbbcf76d616b5870008a3b314f75ede8",
		},
		Spand: "d",
		Span:  1,
		STime: 1537718400,
		ETime: 1538323200,
		/*Group: []string{"typeid",},*/
		Group: []string{"typeid", "s_time"},
	})
	fmt.Println(err)
	data := map[string]interface{}{}
	utils.JsonDecode(res.Data, &data)
	fmt.Println(res.Code)
	fmt.Println(data["num"])
	for _, v := range data["list"].([]interface{}) {
		fmt.Println(v)
	}
}
func send() {
	wgp := sync.WaitGroup{}
	wgp.Add(10)
	no := time.Now()
	for x := 0; x < 10; x++ {
		go func() {
			conn, err := flysnow.NewConnection("192.168.1.90", 22258)
			fmt.Println(err)
			for i := 0; i < 50000; i++ {
				conn.Send("apis", map[string]interface{}{
					"api":    "1234",
					"appkey": "abcd",
					"code":   "1",
				})
				time.Sleep(1 * time.Second)
			}
			wgp.Done()
		}()
	}
	wgp.Wait()
	fmt.Println(time.Since(no))
}
