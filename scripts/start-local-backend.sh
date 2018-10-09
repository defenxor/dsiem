#!/bin/bash
exec ./dsiem serve --dev -m cluster-backend --msq test --msqUrl nats://localhost:4222 --node backend1 --port 8081 --frontend http://localhost:8080
