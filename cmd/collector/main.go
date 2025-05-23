package main

import (
    "fmt"
    "log"
    "os"
    "strconv"
	"path/filepath"

    "github.com/asmit27rai/collector/pkg/collector"
    "github.com/asmit27rai/collector/pkg/writer"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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

        if err := collectCustomResources(its, args, nsName, nsPath); err != nil {
            return err
        }
        
        // Collect custom resources
        if err := collectCustomResources(its, args, nsName, nsPath); err != nil {
            return err
        }
    }
    return nil
}

func collectCustomResources(its *collector.Collector, args collector.CollectionArgs, nsName string, nsPath string) error {
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

    // // Collect AppliedManifestWork
    // appliedGVR := schema.GroupVersionResource{
    //     Group:    "internal.open-cluster-management.io",
    //     Version:  "v1beta1",
    //     Resource: "appliedmanifestworks",
    // }
    // appliedMetrics, err := its.CollectCustomResources(
    //     appliedGVR,
    //     "", // Use WEC context as namespace
    //     "",
    // )
    // if err != nil {
    //     return err
    // }

    // Write results
    writer.WriteWorkMetrics(nsPath, "manifestworks", manifestMetrics)
    writer.WriteWorkMetrics(nsPath, "workstatuses", statusMetrics)
    // writer.WriteWorkMetrics(nsPath, "appliedmanifestworks", appliedMetrics)

    return nil
}