#!/bin/bash
set -a      # turn on automatic exporting
. .env  # source test.env
set +a      # turn off automatic exporting

go build
go run .