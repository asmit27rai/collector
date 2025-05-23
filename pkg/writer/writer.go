package writer

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/asmit27rai/collector/pkg/collector"
)

func WriteMetrics(path, kind, context string, metrics []collector.ObjectMetrics) error {
    dir := filepath.Join(path, kind+"-"+context)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %v", err)
    }

    file := filepath.Join(dir, kind+".csv")
    f, err := os.Create(file)
    if err != nil {
        return fmt.Errorf("failed to create file: %v", err)
    }
    defer f.Close()

    for _, m := range metrics {
        line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", 
            m.Name, m.Created, m.StatusUpdate, m.Condition, m.Manager)
        if _, err := f.WriteString(line); err != nil {
            return fmt.Errorf("failed to write to file: %v", err)
        }
    }
    return nil
}

func WriteWorkMetrics(path, kind string, metrics []collector.WorkMetrics) error {
    dir := filepath.Join(path, kind)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %v", err)
    }

    file := filepath.Join(dir, kind+".csv")
    f, err := os.Create(file)
    if err != nil {
        return fmt.Errorf("failed to create file: %v", err)
    }
    defer f.Close()

    for _, m := range metrics {
        line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", 
            m.Name, m.Created, m.Updated, m.Status, m.TargetObject)
        if _, err := f.WriteString(line); err != nil {
            return fmt.Errorf("failed to write to file: %v", err)
        }
    }
    return nil
}