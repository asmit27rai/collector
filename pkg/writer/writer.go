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
        return err
    }

    file := filepath.Join(dir, kind+".csv")
    f, err := os.Create(file)
    if err != nil {
        return err
    }
    defer f.Close()

    // Write header
    if _, err := f.WriteString("Name\tCreated\tStatusUpdate\tCondition\tManager\n"); err != nil {
        return err
    }

    // Write data
    for _, m := range metrics {
        line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", 
            m.Name, m.Created, m.StatusUpdate, m.Condition, m.Manager)
        if _, err := f.WriteString(line); err != nil {
            return err
        }
    }
    return nil
}

func WriteWorkMetrics(path, kind string, metrics []collector.WorkMetrics) error {
    dir := filepath.Join(path, kind)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    file := filepath.Join(dir, kind+".csv")
    f, err := os.Create(file)
    if err != nil {
        return err
    }
    defer f.Close()

    // Write header
    if _, err := f.WriteString("Name\tCreated\tUpdated\tStatus\tTargetObject\n"); err != nil {
        return err
    }

    // Write data
    for _, m := range metrics {
        line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", 
            m.Name, m.Created, m.Updated, m.Status, m.TargetObject)
        if _, err := f.WriteString(line); err != nil {
            return err
        }
    }
    return nil
}