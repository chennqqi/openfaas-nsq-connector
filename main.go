// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/openfaas-incubator/connector-sdk/types"
)

type connectorConfig struct {
	*types.ControllerConfig
	Topics []string
	Broker string

	//for nsq
	Lookupd     []string
	Nsqd        []string
	MaxInFlight int
}

func main() {
	credentials := types.GetCredentials()
	config := buildConnectorConfig()

	controller := types.NewController(credentials, config.ControllerConfig)

	controller.BeginMapBuilder()

	waitForBrokers(config, controller)
	makeConsumer(config, controller)
}

func waitForBrokers(config connectorConfig, controller *types.Controller) {
	//only check tcp connection
	var client sarama.Client
	var err error

	for {
		if len(config.Lookupd) > 0 {
			for i := 0; i < len(config.Lookupd); i++ {
				conn, err := net.DialTimeout("tcp", config.Lookupd[i], timeout)
				//at least one connect ok
				if err == nil {
					conn.Close()
					return
				}
			}
		} else {
			var expect = true
			for i := 0; i < len(config.Nsqd); i++ {
				conn, err := net.DialTimeout("tcp", config.Lookupd[i], timeout)
				//all should least one connect ok
				if err != nil { //at least one connect ok
					expect = false
					break
				} else {
					conn.Close()
				}
			}
			if expect {
				return
			}
		}
		time.Sleep(1 * time.Second)
	}
}

type NsqHandler struct {
	topic      string
	controller *types.Controller
	consumer   *nsq.Consumer
	num        int
}

func (this *NsqHandler) HandleMessage(message *nsq.Message) error {
	this.num = (this.num + 1) % math.MaxInt32
	fmt.Printf("[#%d] Received on [%v,%v]: '%s'\n",
		num,
		this.Topic,
		message.NSQDAddress,
		string(message.Body))

	controller.Invoke(this.Topic, &message.Body)
	message.Finish() // mark message as processed
}

func makeConsumer(config connectorConfig, controller *types.Controller) {
	//setup nsq consumer
	nsqConfig := nsq.NewConfig()
	nsqConfig.MaxInFlight = config.MaxInFlight
	nsqConfig.WriteTimeout = 6 * time.Second
	nsqConfig.DialTimeout = 4 * time.Second

	group := "faas-nsq-queue-workers"

	topics := config.Topics
	log.Printf("Binding to topics: %v", config.Topics)

	var consumers []*nsq.Consumer
	for i := 0; i < len(config.Topics); i++ {
		consumer, err := nsq.NewConsumer(config.Topics[i], group, nsqConfig)
		if err != nil {
			log.Fatalln("Fail to create nsq consumer: ", err)
		}
		consumer.AddHandler(&NsqHandler{
			config.Topics[i], controller, consumer, 0,
		})

		if len(config.Lookupd) > 0 {
			err = consumer.ConnectToNSQLookupds(config.Lookupd)
		} else {
			err = consumer.ConnectToNSQDs(config.Nsqd)
		}
		if err != nil {
			log.Fatalln("Fail to create Kafka consumer: ", err)
		}
		consumers = append(consumers, consumer)
	}
	//stop all consumer, unuse
	defer func() {
		for i := 0; i < len(consumers); i++ {
			c := consumers[i]
			if c != nil {
				c.Stop()
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(consumer))

	go func() {
		for i := 0; i < len(consumers); i++ {
			c := consumer
			<-c.StopChan
			wg.Done()
		}
	}()

	wg.Wait()
}

func buildConnectorConfig() connectorConfig {
	nsqlookup := "nsqlookupd"
	if val, exists := os.LookupEnv("nsqlookupd"); exists {
		nsqlookup = val
	}
	nsqd := "nsqd"
	if val, exists := os.LookupEnv("nsqd"); exists {
		nsqd = val
	}
	maxInflight := 1000
	if val, exists := os.LookupEnv("nsq_maxinflight"); exists {
		fmt.Sscanf(val, "%d", &maxInflight)
	}

	topics := []string{}
	if val, exists := os.LookupEnv("topics"); exists {
		for _, topic := range strings.Split(val, ",") {
			if len(topic) > 0 {
				topics = append(topics, topic)
			}
		}
	}
	if len(topics) == 0 {
		log.Fatal(`Provide a list of topics i.e. topics="payment_published,slack_joined"`)
	}

	gatewayURL := "http://gateway:8080"
	if val, exists := os.LookupEnv("gateway_url"); exists {
		gatewayURL = val
	}

	upstreamTimeout := time.Second * 30
	rebuildInterval := time.Second * 3

	if val, exists := os.LookupEnv("upstream_timeout"); exists {
		parsedVal, err := time.ParseDuration(val)
		if err == nil {
			upstreamTimeout = parsedVal
		}
	}

	if val, exists := os.LookupEnv("rebuild_interval"); exists {
		parsedVal, err := time.ParseDuration(val)
		if err == nil {
			rebuildInterval = parsedVal
		}
	}

	printResponse := false
	if val, exists := os.LookupEnv("print_response"); exists {
		printResponse = (val == "1" || val == "true")
	}

	printResponseBody := false
	if val, exists := os.LookupEnv("print_response_body"); exists {
		printResponseBody = (val == "1" || val == "true")
	}

	delimiter := ","
	if val, exists := os.LookupEnv("topic_delimiter"); exists {
		if len(val) > 0 {
			delimiter = val
		}
	}

	return connectorConfig{
		ControllerConfig: &types.ControllerConfig{
			UpstreamTimeout:          upstreamTimeout,
			GatewayURL:               gatewayURL,
			PrintResponse:            printResponse,
			PrintResponseBody:        printResponseBody,
			RebuildInterval:          rebuildInterval,
			TopicAnnotationDelimiter: delimiter,
		},
		Topics:      topics,
		Lookupd:     nsqlookup,
		Nsqd:        nsqd,
		MaxInFlight: maxInflight,
	}
}
