#!/bin/bash
ng build --prod --build-optimizer
cd dist
python -m SimpleHTTPServer 8000
