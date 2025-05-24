package main

import (
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "os/exec"
    "strings"
    "time"
    "strconv"
	"path/filepath"

    "github.com/asmit27rai/collector/pkg/collector"
    "github.com/asmit27rai/collector/pkg/writer"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// LatencyData holds timestamps for calculations
type LatencyData struct {
    WDSDeployCreate    time.Time
    WDSDeployStatus    time.Time
    WECDeployCreate    time.Time
    WECDeployStatus    time.Time
    BindingCreate      time.Time
    ManifestWorkCreate time.Time
    AppliedManifestCreate time.Time
    WorkStatusUpdate   time.Time
}

func collectLongExperiment(wds, its, wec *collector.Collector, args collector.CollectionArgs) error {
    // Implement long-running experiment collection
    log.Println("Long experiment collection not implemented yet")
    return nil
}

func main() {
    if len(os.Args) < 7 {
        log.Fatal("Usage: collector <kubeconfig> <wds-context> <its-context> <wec-context> <num-ns> <output-dir> [exp-type]")
    }

    args := parseArgs(os.Args[1:])
    if err := runCollection(args); err != nil {
        log.Fatal(err)
    }
}

func parseArgs(args []string) collector.CollectionArgs {
    numNS, _ := strconv.Atoi(args[4])
    expType := "s"
    if len(args) > 7 {
        expType = args[7]
    }

    // kubeconfig, err := filepath.EvalSymlinks(args[0])
    // if err != nil {
    //     log.Fatalf("Failed to resolve kubeconfig path: %v", err)
    // }

    return collector.CollectionArgs{
        Kubeconfig:  /*kubeconfig*/ args[0],
        WDSContext:  args[1],
        ITSContext:  args[2],
        WECContext:  args[3],
        NumNS:       numNS,
        OutputDir:   args[5],
        ExpType:     expType,
    }
}

func runCollection(args collector.CollectionArgs) error {
    wdsCollector, err := collector.NewCollector(args.Kubeconfig, args.WDSContext)
    if err != nil {
        return err
    }

    itsCollector, err := collector.NewCollector(args.Kubeconfig, args.ITSContext)
    if err != nil {
        return err
    }

    wecCollector, err := collector.NewCollector(args.Kubeconfig, args.WECContext)
    if err != nil {
        return err
    }

    if args.ExpType == "s" {
        return collectShortExperiment(wdsCollector, itsCollector, wecCollector, args)
    }
    return collectLongExperiment(wdsCollector, itsCollector, wecCollector, args)
}

func collectShortExperiment(wds, its, wec *collector.Collector, args collector.CollectionArgs) error {
    objKinds := []string{"deployments", "secrets", "configmaps", "services"}
    
    for ns := 0; ns < args.NumNS; ns++ {
        nsName := fmt.Sprintf("perf-test-%d", ns)
        nsPath := filepath.Join(args.OutputDir, nsName)
        
        // Collect standard resources
        for _, kind := range objKinds {
            // WDS metrics
            wdsMetrics, err := wds.CollectStandardObjects(kind, nsName)
            if err != nil {
                return err
            }
            writer.WriteMetrics(nsPath, kind, "wds", wdsMetrics)

            // WEC metrics
            wecMetrics, err := wec.CollectStandardObjects(kind, nsName)
            if err != nil {
                return err
            }
            writer.WriteMetrics(nsPath, kind, "wec", wecMetrics)
        }

        if err := collectCustomResources(its, wec, args, nsName, nsPath); err != nil {
            return err
        }
        
        // Collect custom resources
        if err := collectCustomResources(its, wec, args, nsName, nsPath); err != nil {
            return err
        }
    }
    latencyData, err := gatherLatencyData(args.OutputDir, args.WDSContext)
    if err != nil {
        return fmt.Errorf("error gathering latency data: %v", err)
    }

    // Write to file instead of terminal
    if err := writeLatenciesToFile(latencyData, args.OutputDir); err != nil {
        return fmt.Errorf("error writing results: %v", err)
    }
    
    log.Printf("✅ Metrics written to: %s/latency_results.txt", args.OutputDir)
    return nil
}

func collectCustomResources(its *collector.Collector, wec *collector.Collector, args collector.CollectionArgs, nsName string, nsPath string) error {
    bindingPolicy := nsName
    labelSelector := fmt.Sprintf("transport.kubestellar.io/originOwnerReferenceBindingKey=%s", bindingPolicy)

    // Collect ManifestWorks
    manifestGVR := schema.GroupVersionResource{
        Group:    "work.open-cluster-management.io",
        Version:  "v1",
        Resource: "manifestworks",
    }
    manifestMetrics, err := its.CollectCustomResources(
        manifestGVR, 
        args.WECContext, // Use WEC context as namespace
        labelSelector,
    )
    if err != nil {
        return err
    }

    // Collect WorkStatuses
    statusGVR := schema.GroupVersionResource{
        Group:    "control.kubestellar.io",
        Version:  "v1alpha1",
        Resource: "workstatuses",
    }
    statusMetrics, err := its.CollectCustomResources(
        statusGVR,
        args.WECContext, // Use WEC context as namespace
        labelSelector,
    )
    if err != nil {
        return err
    }

    // Collect AppliedManifestWork
    appliedGVR := schema.GroupVersionResource{
        Group:    "work.open-cluster-management.io",
        Version:  "v1",
        Resource: "appliedmanifestworks",
    }
    appliedMetrics, err := wec.CollectCustomResources(
        appliedGVR,
        "",
        "",
    )
    if err != nil {
        fmt.Printf("Error collecting applied manifest works: %v\n", err)
        return err
    }

    // Write results
    writer.WriteWorkMetrics(nsPath, "manifestworks", manifestMetrics)
    writer.WriteWorkMetrics(nsPath, "workstatuses", statusMetrics)
    writer.WriteWorkMetrics(nsPath, "appliedmanifestworks", appliedMetrics)

    return nil
}

func gatherLatencyData(outputDir, wdsContext string) (*LatencyData, error) {
    data := &LatencyData{}
    var err error

    log.Println("Gathering latency data...")
    
    // Get binding policy creation time
    data.BindingCreate, err = getBindingCreationTime(wdsContext)
    if err != nil {
        return nil, fmt.Errorf("failed to get binding creation time: %v", err)
    }
    log.Printf("Binding created at: %v", data.BindingCreate)

    // Read WDS deployment timestamps
    wdsPath := filepath.Join(outputDir, "perf-test-0/deployments-wds/deployments.csv")
    log.Printf("Reading WDS deployments from: %s", wdsPath)
    data.WDSDeployCreate, data.WDSDeployStatus, err = readDeploymentTimestamps(wdsPath)
    if err != nil {
        return nil, fmt.Errorf("error reading WDS deployments: %v", err)
    }

    // Read WEC deployment timestamps
    wecPath := filepath.Join(outputDir, "perf-test-0/deployments-wec/deployments.csv")
    log.Printf("Reading WEC deployments from: %s", wecPath)
    data.WECDeployCreate, data.WECDeployStatus, err = readDeploymentTimestamps(wecPath)
    if err != nil {
        return nil, fmt.Errorf("error reading WEC deployments: %v", err)
    }

    // Read ManifestWork timestamps
    mwPath := filepath.Join(outputDir, "perf-test-0/manifestworks/manifestworks.csv")
    log.Printf("Reading ManifestWorks from: %s", mwPath)
    data.ManifestWorkCreate, err = readCSVTimestamp(mwPath, 1)
    if err != nil {
        return nil, fmt.Errorf("error reading ManifestWorks: %v", err)
    }

    // Read AppliedManifestWork timestamps
    amwPath := filepath.Join(outputDir, "perf-test-0/appliedmanifestworks/appliedmanifestworks.csv")
    log.Printf("Reading AppliedManifestWorks from: %s", amwPath)
    data.AppliedManifestCreate, err = readCSVTimestamp(amwPath, 1)
    if err != nil {
        return nil, fmt.Errorf("error reading AppliedManifestWorks: %v", err)
    }

    // Handle WorkStatus with fallback
    wsPath := filepath.Join(outputDir, "perf-test-0/workstatuses/workstatuses.csv")
    log.Printf("Reading WorkStatuses from: %s", wsPath)
    data.WorkStatusUpdate, err = readCSVTimestamp(wsPath, 1)
    if err != nil {
        log.Printf("WorkStatus update time unavailable: %v", err)
        log.Println("   This is normal if status hasn't been reported yet")
        data.WorkStatusUpdate = time.Time{} // Explicit zero time
    }

    log.Println("All latency data collected successfully")
    return data, nil
}

func getBindingCreationTime(context string) (time.Time, error) {
    // Get actual binding policy resource (not CRD)
    cmd := exec.Command("kubectl", "--context", context, "get", "bindingpolicies.control.kubestellar.io", 
        "nginx-bpolicy", "-o", "jsonpath={.metadata.creationTimestamp}")
    
    output, err := cmd.Output()
    if err != nil {
        return time.Time{}, fmt.Errorf("error getting binding policy: %v\nDid you create the binding policy after the deployment?", err)
    }

    return time.Parse(time.RFC3339, strings.TrimSpace(string(output)))
}

func readDeploymentTimestamps(path string) (time.Time, time.Time, error) {
    file, err := os.Open(path)
    if err != nil {
        return time.Time{}, time.Time{}, err
    }
    defer file.Close()

    reader := csv.NewReader(file)
    reader.Comma = '\t'
    reader.FieldsPerRecord = -1 // Allow variable fields
    reader.LazyQuotes = true

    // Skip header
    if _, err := reader.Read(); err != nil {
        return time.Time{}, time.Time{}, err
    }

    records, err := reader.ReadAll()
    if err != nil {
        return time.Time{}, time.Time{}, err
    }

    if len(records) < 1 {
        return time.Time{}, time.Time{}, fmt.Errorf("no data rows in %s", path)
    }

    dataRow := records[0] // First data row after header
    if len(dataRow) < 4 {
        return time.Time{}, time.Time{}, fmt.Errorf("invalid data format in %s", path)
    }

    creationTime, err := time.Parse(time.RFC3339, dataRow[1])
    if err != nil {
        return time.Time{}, time.Time{}, err
    }

    statusTime, err := time.Parse(time.RFC3339, dataRow[2])
    if err != nil {
        return time.Time{}, time.Time{}, err
    }

    return creationTime, statusTime, nil
}

func readCSVTimestamp(path string, timeColumn int) (time.Time, error) {
    file, err := os.Open(path)
    if err != nil {
        return time.Time{}, fmt.Errorf("failed to open file: %v", err)
    }
    defer file.Close()

    reader := csv.NewReader(file)
    reader.Comma = '\t'
    reader.FieldsPerRecord = -1
    reader.LazyQuotes = true

    // Skip header
    if _, err := reader.Read(); err != nil {
        return time.Time{}, fmt.Errorf("header read error: %v", err)
    }

    records, err := reader.ReadAll()
    if err != nil {
        return time.Time{}, fmt.Errorf("data read error: %v", err)
    }

    if len(records) == 0 {
        return time.Time{}, fmt.Errorf("no data rows found in %s", path)
    }

    dataRow := records[0] // First data row
    if len(dataRow) <= timeColumn {
        return time.Time{}, fmt.Errorf("column %d missing in %s", timeColumn, path)
    }

    timeStr := strings.TrimSpace(dataRow[timeColumn])
    if timeStr == "" {
        return time.Time{}, fmt.Errorf("empty timestamp in column %d of %s", 
            timeColumn, filepath.Base(path))
    }

    parsedTime, err := time.Parse(time.RFC3339, timeStr)
    if err != nil {
        return time.Time{}, fmt.Errorf("invalid time format in %s: %v", 
            filepath.Base(path), err)
    }

    return parsedTime, nil
}

func calculateAndPrintLatencies(data *LatencyData) {

    fmt.Println("\n ====== KubeStellar Performance Results ======")
    
    // Downsync metrics
    fmt.Println("\n Downsync Metrics")
    fmt.Printf("  Binding Creation: %v\n", data.BindingCreate.Sub(data.WDSDeployCreate).Round(time.Millisecond))
    fmt.Printf("  Packaging Time:   %v\n", data.ManifestWorkCreate.Sub(data.BindingCreate).Round(time.Millisecond))
    fmt.Printf("  Delivery Time:    %v\n", data.AppliedManifestCreate.Sub(data.ManifestWorkCreate).Round(time.Millisecond))
    fmt.Printf("  Activation Time:  %v\n", data.WECDeployCreate.Sub(data.AppliedManifestCreate).Round(time.Millisecond))
    
    // Upsync metrics
    fmt.Println("\n Upsync Metrics")
    fmt.Printf("  Status Report:    %v\n", data.WorkStatusUpdate.Sub(data.WECDeployStatus).Round(time.Millisecond))
    fmt.Printf("  Finalization:     %v\n", data.WDSDeployStatus.Sub(data.WorkStatusUpdate).Round(time.Millisecond))
    
    // Totals
    fmt.Println("\n Totals")
    fmt.Printf("  Total Downsync:   %v\n", data.WECDeployCreate.Sub(data.WDSDeployCreate).Round(time.Millisecond))
    fmt.Printf("  Total Upsync:     %v\n", data.WDSDeployStatus.Sub(data.WECDeployStatus).Round(time.Millisecond))
    fmt.Printf("  End-to-End:       %v\n", data.WDSDeployStatus.Sub(data.WDSDeployCreate).Round(time.Millisecond))
    
    fmt.Println("===========================================")
}

func writeLatenciesToFile(data *LatencyData, outputDir string) error {
    resultsPath := filepath.Join(outputDir, "latency_results.txt")
    file, err := os.Create(resultsPath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Helper function to format durations safely
    formatDuration := func(d time.Duration) string {
        if d < 0 {
            return "N/A (invalid timestamp order)"
        }
        return d.Round(time.Millisecond).String()
    }

    content := fmt.Sprintf(`KubeStellar Performance Metrics
================================
Downsync Metrics
----------------
Binding Creation:    %s
Packaging Time:      %s
Delivery Time:       %s
Activation Time:     %s
Total Downsync:      %s

Upsync Metrics
--------------
Status Report:       %s
Finalization Time:   %s
Total Upsync:        %s

End-to-End Latency
------------------
Total Lifecycle:     %s`,
        formatDuration(data.BindingCreate.Sub(data.WDSDeployCreate)),
        formatDuration(data.ManifestWorkCreate.Sub(data.BindingCreate)),
        formatDuration(data.AppliedManifestCreate.Sub(data.ManifestWorkCreate)),
        formatDuration(data.WECDeployCreate.Sub(data.AppliedManifestCreate)),
        formatDuration(data.WECDeployCreate.Sub(data.WDSDeployCreate)),
        formatDuration(data.WorkStatusUpdate.Sub(data.WECDeployStatus)),
        formatDuration(data.WDSDeployStatus.Sub(data.WorkStatusUpdate)),
        formatDuration(data.WDSDeployStatus.Sub(data.WECDeployStatus)),
        formatDuration(data.WDSDeployStatus.Sub(data.WDSDeployCreate)),
    )

    _, err = file.WriteString(content)
    return err
}
