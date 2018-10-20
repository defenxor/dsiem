#!/bin/bash
exec ./dsiem serve -m cluster-frontend --msq nats://localhost:4222/ --node frontend1 -e 2000

