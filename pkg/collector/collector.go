package collector

import (
    "context"
    "fmt"
    "time"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Collector struct {
    Clientset *kubernetes.Clientset
    Context   string
}

func parseServiceMetrics(svc corev1.Service) ObjectMetrics {
    return ObjectMetrics{
        Name:         svc.Name,
        Namespace:    svc.Namespace,
        Created:      svc.CreationTimestamp.Format(time.RFC3339),
        StatusUpdate: getStatusTime(svc.ObjectMeta),
        Condition:    "Active",  // Add proper status detection
        Manager:      getManager(svc.ObjectMeta),
    }
}

func parseSecretMetrics(secret corev1.Secret) ObjectMetrics {
    return ObjectMetrics{
        Name:         secret.Name,
        Namespace:    secret.Namespace,
        Created:      secret.CreationTimestamp.Format(time.RFC3339),
        StatusUpdate: "",  // Secrets typically don't have status
        Condition:    "Exists",
        Manager:      getManager(secret.ObjectMeta),
    }
}

func parseConfigMapMetrics(cm corev1.ConfigMap) ObjectMetrics {
    return ObjectMetrics{
        Name:         cm.Name,
        Namespace:    cm.Namespace,
        Created:      cm.CreationTimestamp.Format(time.RFC3339),
        StatusUpdate: "",  // ConfigMaps typically don't have status
        Condition:    "Exists",
        Manager:      getManager(cm.ObjectMeta),
    }
}

func NewCollector(kubeconfig, contextName string) (*Collector, error) {
    config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
        &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
        &clientcmd.ConfigOverrides{CurrentContext: contextName},
    ).ClientConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to create client config: %v", err)
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create clientset: %v", err)
    }

    return &Collector{
        Clientset: clientset,
        Context:   contextName,
    }, nil
}

func (c *Collector) CollectStandardObjects(kind, namespace string) ([]ObjectMetrics, error) {
    var metrics []ObjectMetrics

    switch kind {
    case "deployments":
        deps, err := c.Clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
        if err != nil {
            return nil, err
        }
        for _, dep := range deps.Items {
            metrics = append(metrics, parseDeploymentMetrics(dep))
        }
    case "services":
        svcs, err := c.Clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
        if err != nil {
            return nil, err
        }
        for _, svc := range svcs.Items {
            metrics = append(metrics, parseServiceMetrics(svc))
        }
    case "secrets":
        secrets, err := c.Clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
        if err != nil {
            return nil, err
        }
        for _, secret := range secrets.Items {
            metrics = append(metrics, parseSecretMetrics(secret))
        }
    case "configmaps":
        cms, err := c.Clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
        if err != nil {
            return nil, err
        }
        for _, cm := range cms.Items {
            metrics = append(metrics, parseConfigMapMetrics(cm))
        }
    }

    return metrics, nil
}

func parseDeploymentMetrics(dep appsv1.Deployment) ObjectMetrics {
    condition := "Unavailable"
    if dep.Status.ReadyReplicas == *dep.Spec.Replicas {
        condition = "Available"
    }
    
    return ObjectMetrics{
        Name:         dep.Name,
        Namespace:    dep.Namespace,
        Created:      dep.CreationTimestamp.Format(time.RFC3339),
        StatusUpdate: getStatusTime(dep.ObjectMeta),
        Condition:    condition,
        Manager:      getManager(dep.ObjectMeta),
    }
}

func getStatusTime(meta metav1.ObjectMeta) string {
    for _, mf := range meta.ManagedFields {
        if mf.Operation == "Update" && mf.Subresource == "status" {
            return mf.Time.Format(time.RFC3339)
        }
    }
    return ""
}

func getManager(meta metav1.ObjectMeta) string {
    managers := []string{"kube-controller-manager", "controller-manager", "kubelet"}
    for _, mf := range meta.ManagedFields {
        for _, m := range managers {
            if mf.Manager == m {
                return m
            }
        }
    }
    return ""
}