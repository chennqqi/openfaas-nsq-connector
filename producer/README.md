This producer can be used to generate load for the Kafka connector.

Example:

* Ingest 15k messages 100ms apart:

```shell
$ kubectl delete deploy/producer -n openfaas

$ kubectl run -n openfaas -t -i --image alexellis2/kafka-producer:0.2.0 producer -- /bin/sh

producer -messages=15000 -broker=kafka:9092 -pause 100ms -topic=vm.powered.on
```
