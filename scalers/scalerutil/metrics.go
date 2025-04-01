package util

func GetPodMetrics(clientset *metrics_client.Clientset) (return type??) {
	podMetricsList, err := clientset.MetricsV1beta1().PodMetricses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil { panic(err) }

	return podMetricsList
}