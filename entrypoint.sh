#!/bin/bash

cd glslsandbox
export DEV ADDR AUTH_SECRET IMPORT
export TLS_ADDR DOMAINS
DATA_PATH=/data ./glslsandbox
