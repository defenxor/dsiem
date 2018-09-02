#!/bin/bash
app="icinga-webhook-ssh"
ver="1.0.0"

docker build -f Dockerfile -t defenxor/${app}:${ver} -t defenxor/${app}:latest . 
