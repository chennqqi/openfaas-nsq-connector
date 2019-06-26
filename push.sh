#!/bin/sh
set -e

if [ -z "$NAMESPACE" ]; then
    NAMESPACE="openfaas"
fi

docker push $NAMESPACE/kafka-connector:$TAG

