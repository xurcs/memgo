package gui

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

var (
	monitorLabel  = color.New(color.FgGreen, color.Bold).Sprint("[MONITOR]")
	cleanedLabel  = color.New(color.FgGreen, color.Bold).Sprint("[CLEANED]")
	infoLabel     = color.New(color.FgCyan, color.Bold).Sprint("[INFO]")
	autoLabel     = color.New(color.FgYellow, color.Bold).Sprint("[AUTOCLEANER]")
	errorLabel    = color.New(color.FgRed, color.Bold).Sprint("[ERROR]")
	warnColor     = color.New(color.FgRed, color.Bold)
	normalColor   = color.New(color.FgWhite)
	lastUsage     = -1.0
	lastPrintTime = time.Time{}
)

func timestamp() string {
	return time.Now().Format("[15:04:05]")
}

func PrintInfo(format string, args ...interface{}) {
	fmt.Printf("%s %s %s\n", timestamp(), infoLabel, fmt.Sprintf(format, args...))
}

func PrintMonitor(format string, args ...interface{}) {
	fmt.Printf("%s %s %s\n", timestamp(), monitorLabel, fmt.Sprintf(format, args...))
}

func PrintCleaned(format string, args ...interface{}) {
	fmt.Printf("%s %s %s\n", timestamp(), cleanedLabel, fmt.Sprintf(format, args...))
}

func PrintAuto(format string, args ...interface{}) {
	fmt.Printf("%s %s %s\n", timestamp(), autoLabel, fmt.Sprintf(format, args...))
}

func PrintError(format string, args ...interface{}) {
	fmt.Printf("%s %s %s\n", timestamp(), errorLabel, fmt.Sprintf(format, args...))
}

func PrintMemoryStats(usedPercent, availableGB, totalGB float64, threshold int) {
	now := time.Now()

	shouldPrint := lastUsage < 0 ||
		abs(usedPercent-lastUsage) > 1.0 ||
		(usedPercent >= float64(threshold)) != (lastUsage >= float64(threshold)) ||
		now.Sub(lastPrintTime) >= 1*time.Second

	if !shouldPrint {
		return
	}

	lastUsage = usedPercent
	lastPrintTime = now

	fmt.Printf("%s %s Memory: ", timestamp(), infoLabel)
	if usedPercent >= float64(threshold) {
		warnColor.Printf("%.1f%% used", usedPercent)
	} else {
		normalColor.Printf("%.1f%% used", usedPercent)
	}
	fmt.Printf(" | %.2f GB available | %.2f GB total\n", availableGB, totalGB)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func PrintConfig(cleanThreshold, cleanInterval int) {
	intervalText := "Disabled"
	if cleanInterval > 0 {
		intervalText = fmt.Sprintf("Every %d minutes", cleanInterval)
	}

	fmt.Printf("\n%s MEMGO - Memory Cleaner\n", monitorLabel)
	fmt.Printf("%s Configuration:\n", autoLabel)
	fmt.Printf("  • Clean threshold: %d%%\n", cleanThreshold)
	fmt.Printf("  • Auto-clean interval: %s\n\n", intervalText)
}

func PrintCleaningStart(threshold, currentUsage float64) {
	fmt.Printf("%s %s Memory usage above threshold (%.0f%%)\n", timestamp(), autoLabel, threshold)
	fmt.Printf("  • Current usage: %.1f%%\n", currentUsage)
	fmt.Printf("  • Cleaning RAM...\n")
}

func PrintCleaningComplete() {
	fmt.Printf("%s %s RAM cleaned successfully\n", timestamp(), cleanedLabel)
}
