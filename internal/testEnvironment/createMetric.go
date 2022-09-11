package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"

	"github.com/GermanVor/devops-pet-project/cmd/agent/utils"
	"github.com/GermanVor/devops-pet-project/internal/common"
)

func main() {
	mType := ""

	var req *http.Request
	var err error

	for {
		fmt.Fscan(os.Stdin, &mType)

		switch mType {
		case "g":
			value := rand.Float64()
			req, err = utils.BuildRequestV2("http://localhost:8080", common.GaugeMetricName, "qwertyG", fmt.Sprintf("%v", value), "")
		case "c":
			req, err = utils.BuildRequestV2("http://localhost:8080", common.CounterMetricName, "qwertyC", "5", "")
		default:
			log.Println("unknown type")
			continue
		}

		if err != nil {
			log.Fatalln(err.Error())
		}

		log.Println(req.Body)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalln(err.Error())
		}

		log.Println(res)
		res.Body.Close()
	}
}
