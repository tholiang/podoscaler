//go:build autoscalertest
// +build autoscalertest

package main

import (
	test "github.com/tholiang/podoscaler/scalers/autoscalertest"
)

func main() {
	test.RunIntegrationTests()
}
