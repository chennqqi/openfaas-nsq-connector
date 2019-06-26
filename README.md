## Nsq-connector

[![Go Report Card](https://goreportcard.com/badge/github.com/chennqqi/openfaas-nsq-connector)](https://goreportcard.com/badge/github.com/chennqqi/openfaas-nsq-connector) [![Build
Status](https://travis-ci.org/chennqqi/openfaas-nsq-connector.svg?branch=master)](https://travis-ci.org/chennqqi/openfaas-nsq-connector) [![GoDoc](https://godoc.org/github.com/chennqqi/openfaas-nsq-connector?status.svg)](https://godoc.org/github.com/chennqqi/openfaas-nsq-connector) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)

The Nsq connector connects OpenFaaS functions to Nsq topics.

WIP: This project changed/inspired by [https://github.com/openfaas-incubator/kafka-connector](https://github.com/openfaas-incubator/kafka-connector)


Goals:

* Allow functions to subscribe to Nsq topics
* Ingest data from Nsq and execute functions
* Work with the OpenFaaS REST API / Gateway

## Conceptual design

![](./images/overview.svg)

This diagram shows the Nsq connector on the left hand side. It is responsible for querying the API Gateway for a list of functions. It will then build up a map or table of which functions have advertised an interested in which topics.

When the connector hears a message on an advertised topic it will look that up in the reference table and find out which functions it needs to invoke. Functions are invoked only once and there is no re-try mechanism. The result is printed to the logs of the Nsq connector process.

The cache or list of functions <-> topics is refreshed on a periodic basis.

## Building

```
export TAG=0.1.0
make build push
```

## Try it out

### Deploy on Kubernetes

The following instructions show how to run `Nsq-connector` on Kubernetes.

Deploy a function with a `topic` annotation:

```bash
$ faas store deploy figlet --annotation topic="faas-request" --gateway <faas-netes-gateway-url>
```

Deploy Nsq:

You can run the zookeeper, Nsq-broker and Nsq-connector pods with:

```bash
kubectl apply -f ./yaml/kubernetes/
```

If you already have Nsq then update `./yaml/kubernetes/connector-dep.yml` with your Nsq broker address and then deploy only that file:

```bash
kubectl apply -f ./yaml/kubernetes/connector-dep.yml
```

Alternatively you can use the [Nsq-connector helm chart](https://github.com/openfaas/faas-netes/tree/master/chart/Nsq-connector)

Then use the broker to send messages to the topic:

```bash
BROKER=$(kubectl get pods -n openfaas -l component=Nsq-broker -o name|cut -d'/' -f2)
kubectl exec -n openfaas -t -i $BROKER -- /opt/Nsq_2.12-0.11.0.1/bin/Nsq-console-producer.sh --broker-list Nsq:9092 --topic faas-request

hello world
```
Once you have connected, each new line will be a message published.

If you have an error
```
error: unable to upgrade connection: container not found ("Nsq")
```
just wait and retry.

You can verify the proper path of the publisher script by getting the shell of the running broker:
```
$ kubectl exec -n openfaas -t -i $BROKER -- sh 
/ # find | grep producer

./opt/Nsq_2.12-0.11.0.1/config/producer.properties
./opt/Nsq_2.12-0.11.0.1/bin/Nsq-verifiable-producer.sh
./opt/Nsq_2.12-0.11.0.1/bin/Nsq-producer-perf-test.sh
./opt/Nsq_2.12-0.11.0.1/bin/Nsq-console-producer.sh
```

With the `$BROKER` variable still set, view the list of messages on a given topic:

```
$ kubectl exec -n openfaas -t -i $BROKER -- /opt/Nsq_2.12-0.11.0.1/bin/Nsq-console-consumer.sh --bootstrap-server Nsq:9092 --topic faas-request --from-beginning
```

Now check the connector logs to see the figlet function was invoked:

```bash
CONNECTOR=$(kubectl get pods -n openfaas -o name|grep Nsq-connector|cut -d'/' -f2)
kubectl logs -n openfaas -f --tail 100 $CONNECTOR

2018/08/08 16:54:35 Binding to topics: [faas-request]
2018/08/08 16:54:38 Syncing topic map
Rebalanced: &{Type:rebalance start Claimed:map[] Released:map[] Current:map[]}
Rebalanced: &{Type:rebalance OK Claimed:map[faas-request:[0]] Released:map[] Current:map[faas-request:[0]]}

2018/08/08 16:54:41 Syncing topic map
2018/08/08 16:54:44 Syncing topic map
2018/08/08 16:54:47 Syncing topic map

[#53753] Received on [faas-request,0]: 'hello world.'
2018/08/08 16:57:41 Invoke function: figlet
2018/08/08 16:57:42 Response [200] from figlet  
 _          _ _                            _     _ 
| |__   ___| | | ___   __      _____  _ __| | __| |
| '_ \ / _ \ | |/ _ \  \ \ /\ / / _ \| '__| |/ _` |
| | | |  __/ | | (_) |  \ V  V / (_) | |  | | (_| |
|_| |_|\___|_|_|\___/    \_/\_/ \___/|_|  |_|\__,_|
                                                   
```

### Deploy on Swarm

Deploy the stack which contains Nsq and the connector:

```bash
docker stack deploy Nsq -c ./yaml/connector-swarm.yml
```

* Deploy or update a function so it has an annotation `topic=faas-request` or some other topic

As an example:

```shell
$ faas store deploy figlet --annotation topic="faas-request"
```

The function can advertise more than one topic by using a comma-separated list i.e. `topic=topic1,topic2,topic3`

* Publish some messages to the topic in question i.e. `faas-request`

Instructions are below for publishing messages

* Watch the logs of the Nsq-connector

## Trigger a function via a topic with `exec`

You can use the Nsq container to send a message to the topic.

```
SERVICE_NAME=Nsq_Nsq
TASK_ID=$(docker service ps --filter 'desired-state=running' $SERVICE_NAME -q)
CONTAINER_ID=$(docker inspect --format '{{ .Status.ContainerStatus.ContainerID }}' $TASK_ID)
docker exec -it $CONTAINER_ID Nsq-console-producer --broker-list Nsq:9092 --topic faas-request

hello world
```

## Generate load on the topic

You can use the sample application in the producer folder to generate load for a topic.

Make sure you have some functions advertising an interest in that topic so that they receive the data.

> Note: the producer *must* run inside the Kubernetes or Swarm cluster in order to be able to access the broker(s).

## Monitor the results

Once you have generated some requests or start a load-test you can watch the function invocation rate increasing in Prometheus or watch the logs of the container.

### Prometheus

You can open the Prometheus metrics or Grafana dashboard for OpenFaaS to see the functions being invoked.

### Watch the logs

```
docker service logs openfaas-nsq-connector -f


```


> Note: If the broker has a different name from `Nsq` you can pass the `broker_host` environmental variable. This exclude the port.

## Configuration

This configuration can be set in the YAML files for Kubernetes or Swarm.

| env_var               | description                                                 |
| --------------------- |----------------------------------------------------------   |
| `upstream_timeout`      | Go duration - maximum timeout for upstream function call    |
| `rebuild_interval`      | Go duration - interval for rebuilding function to topic map |
| `topics`                | Topics to which the connector will bind                     |
| `gateway_url`           | The URL for the API gateway i.e. http://gateway:8080 or http://gateway.openfaas:8080 for Kubernetes       |
| `nsqd`           | Default is `nsq`                                          |
| `nslookupd`           | Default is `nslookupd`                                          |
| `print_response`        | Default is `true` - this will output information about the response of calling a function in the logs, including the HTTP status, topic that triggered invocation, the function name, and the length of the response body in bytes |
| `print_response_body`   | Default is `true` - this will print the body of the response of calling a function to stdout |
| `topic_delimiter`   | Default is `,` - Specifies character upon which to split the `topic` annotation when subscribing a function to mulitple topics |

## TODO:

* update document
* add yaml for docker swarm
* dep add vendor 