package collector

type ObjectMetrics struct {
    Name          string
    Namespace     string
    Created       string
    StatusUpdate  string
    Condition     string
    Manager       string
}

type WorkMetrics struct {
    Name         string
    Namespace    string
    Created      string
    Updated      string
    Status       string
    TargetObject string
}

type CollectionArgs struct {
    Kubeconfig  string
    WDSContext  string
    ITSContext  string
    WECContext  string
    NumNS       int
    OutputDir   string
    ExpType     string
    NumPods     int
    WatchSec    int
}