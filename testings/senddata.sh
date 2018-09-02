#!/bin/bash
echo sending data
curl -XPOST http://localhost:8080 -d'{"timestamp": "2018-01-01", "sensor":"sensor1","plugin_id":1001}'


