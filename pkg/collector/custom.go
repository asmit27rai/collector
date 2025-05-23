package collector

import (
    "context"
	"time"
	"strings"

    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/tools/clientcmd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func findTargetObject(item unstructured.Unstructured) string {
    manifests, found, _ := unstructured.NestedSlice(item.Object, "spec", "workload", "manifests")
    if found && len(manifests) > 0 {
        if firstManifest, ok := manifests[0].(map[string]interface{}); ok {
            if name, found, _ := unstructured.NestedString(firstManifest, "metadata", "name"); found {
                return name
            }
        }
    }
    return ""
}

func (c *Collector) CollectCustomResources(gvr schema.GroupVersionResource, namespace, labelSelector string) ([]WorkMetrics, error) {
    dynClient, err := getDynamicClient(c.Context)
    if err != nil {
        return nil, err
    }

    list, err := dynClient.Resource(gvr).Namespace(namespace).List(
        context.TODO(), 
        metav1.ListOptions{
            LabelSelector: labelSelector,
        },
    )
    if err != nil {
        return nil, err
    }

    var metrics []WorkMetrics
    for _, item := range list.Items {
        metrics = append(metrics, parseWorkMetrics(item, gvr))
    }
    return metrics, nil
}

func parseWorkMetrics(item unstructured.Unstructured, gvr schema.GroupVersionResource) WorkMetrics {
    status, _, _ := unstructured.NestedString(item.Object, "status", "phase")
    var targetObj string
    
    switch gvr.Resource {
    case "manifestworks":
        if manifests, found, _ := unstructured.NestedSlice(item.Object, "spec", "workload", "manifests"); found {
            for _, m := range manifests {
                if manifest, ok := m.(map[string]interface{}); ok {
                    if name, _, _ := unstructured.NestedString(manifest, "metadata", "name"); name != "" {
                        targetObj = name
                        break
                    }
                }
            }
        }
    case "workstatuses":
        targetObj = strings.TrimPrefix(item.GetName(), "v1-pod-")
    }

    return WorkMetrics{
        Name:         item.GetName(),
        Namespace:    item.GetNamespace(),
        Created:      item.GetCreationTimestamp().Format(time.RFC3339),
        Status:       status,
        TargetObject: targetObj,
    }
}

func getDynamicClient(contextName string) (dynamic.Interface, error) {
    config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
        clientcmd.NewDefaultClientConfigLoadingRules(),
        &clientcmd.ConfigOverrides{CurrentContext: contextName},
    ).ClientConfig()
    if err != nil {
        return nil, err
    }

    return dynamic.NewForConfig(config)
}