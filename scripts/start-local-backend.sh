#!/bin/bash
exec go run ./cmd/dsiem serve --dev -m cluster-backend --msq nats://localhost:4222 --node backend1 --port 8081 --frontend http://localhost:8080 --apm
