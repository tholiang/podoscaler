//go:build watcher
// +build watcher

package main

import (
	"time"

	"github.com/tholiang/podoscaler/scalers/util"
	watcher "github.com/tholiang/podoscaler/scalers/watcher"
)

func run_autoscaler() {
	w := watcher.Watcher{
		PrometheusUrl: util.DEFAULT_PROMETHEUS_URL,
	}
	err := w.Init()
	if err != nil {
		panic(err)
	}

	for {
		err := w.WatchRound()
		if err != nil {
			panic(err)
		}

		time.Sleep(60 * time.Second)
	}
}

func main() {
	run_autoscaler()
}
