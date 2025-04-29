#!/bin/bash
cd ./scalers/
go test ./autoscaler/autoscaler-unit_test.go  ./autoscaler/autoscaler-interface.go ./autoscaler/autoscaler.go ./autoscaler/autoscaler-metrics.go
cd ..