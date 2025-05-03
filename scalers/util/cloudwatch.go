package util

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube_client "k8s.io/client-go/kubernetes"
)

func GetLatencyCloudwatch(loadbalancer_name string) (map[string]float64, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	client := cloudwatch.NewFromConfig(cfg)

	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Minute) // Replace with your actual start time

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/ELB"),
		MetricName: aws.String("Latency"),
		Dimensions: []types.Dimension{
			{
				Name:  aws.String("LoadBalancerName"),
				Value: aws.String(loadbalancer_name),
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

func GetLoadBalancerName(clientset kube_client.Interface, namespace string, serviceName string) (string, error) {
	svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return "", fmt.Errorf("service is not of type LoadBalancer")
	}

	ingress := svc.Status.LoadBalancer.Ingress
	if len(ingress) == 0 {
		return "", fmt.Errorf("no ingress assigned yet")
	}

	if ingress[0].Hostname == "" {
		return "", fmt.Errorf("no hostname")
	}
	parts := strings.Split(ingress[0].Hostname, "-")
	return parts[0], nil
}
