package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/nsqio/go-nsq"
)

func main() {
	var (
		messages int
		pause    time.Duration
		broker   string
		topic    string
	)

	flag.IntVar(&messages, "messages", 1, "specify the number of messages")
	flag.StringVar(&broker, "broker", "", "nsqd address and port")
	flag.StringVar(&topic, "topic", "faas-request", "topic to produce messages on")
	flag.DurationVar(&pause, "pause", time.Millisecond*100, "pause in Golang duration format")
	flag.Parse()

	nsqConfig := nsq.NewConfig()
	nsqConfig.WriteTimeout = 3 * time.Second
	nsqConfig.DialTimeout = 4 * time.Second

	fmt.Println("Creating producer")
	producer, err := nsq.NewProducer(broker, nsqConfig)
	if err != nil {
		panic(err)
	}

	defer producer.Stop()

	log.Printf("Sending %d messages.\n", messages)
	for i := 0; i < messages; i++ {
		err := producer.Publish(topic, []byte("Test the function."))
		if err != nil {
			panic(err)
		}

		log.Printf("Msg: topic(%s) publish OK\n", topic)
		time.Sleep(pause)
	}
}
