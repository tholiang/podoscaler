package util

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func GetLatencyCloudwatch() (map[string]float64, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	client := cloudwatch.NewFromConfig(cfg)

	lbName := "acb0aebe9e45b4b3fa1f8896fca3c943"
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Minute) // Replace with your actual start time

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/ELB"),
		MetricName: aws.String("Latency"),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("LoadBalancerName"),
				Value: aws.String(lbName),
			},
		},
		StartTime:          aws.Time(startTime),
		EndTime:            aws.Time(endTime),
		Period:             aws.Int32(60),
		ExtendedStatistics: []string{"p90", "p95", "p99", "p99.9", "p99.99", "p99.999", "p100"},
	}

	result, err := client.GetMetricStatistics(context.TODO(), input)
	if err != nil {
		panic("failed to get metrics, " + err.Error())
	}

	for _, dp := range result.Datapoints {
		return dp.ExtendedStatistics, nil
	}
	return nil, fmt.Errorf("No datapoints")
}
