#!/bin/bash

keytool -import -file keys/tls.crt -alias firstCA -keystore test.jks

