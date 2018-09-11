#!/bin/bash

rsync -avz --exclude "siem" --exclude "ossim-directive-converter" --exclude "logs" --exclude "ui" ./* ../src/siem/
