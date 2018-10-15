#!/bin/bash

# filebeat configuration file
kubectl delete configmap dsiem-filebeat-config >/dev/null 2>&1
kubectl create configmap dsiem-filebeat-config --from-file=./configs/filebeat

# apm configuration file
kubectl delete configmap dsiem-apm-config >/dev/null 2>&1
kubectl create configmap dsiem-apm-config --from-file=./configs/apm-server

