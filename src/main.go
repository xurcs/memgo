package main

import (
	"bufio"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"memgo/src/core/gui"
	"memgo/src/core/memory"
	"memgo/src/core/monitor"

	"github.com/pelletier/go-toml/v2"
)

type AutoCleanerConfig struct {
	CleanAbove    int `toml:"CLEAN_ABOVE"`
	CleanInterval int `toml:"CLEAN_INTERVAL"`
}

type GeneralConfig struct {
	UpdateInterval float64 `toml:"UPDATE_INTERVAL"`
}

type FullConfig struct {
	AutoCleaner AutoCleanerConfig `toml:"AUTOCLEANER"`
	Config      GeneralConfig     `toml:"CONFIG"`
}

func loadConfig() (*AutoCleanerConfig, float64, error) {
	data, err := os.ReadFile("Memgo.toml")
	if err != nil {
		if os.IsNotExist(err) {
			return &AutoCleanerConfig{90, 0}, 0.5, nil
		}
		return nil, 0, err
	}

	var fullConfig FullConfig
	if err := toml.Unmarshal(data, &fullConfig); err != nil {
		return nil, 0, err
	}

	updateInterval := fullConfig.Config.UpdateInterval
	if updateInterval <= 0 {
		updateInterval = 0.5
	}

	return &fullConfig.AutoCleaner, updateInterval, nil
}

func main() {
	autoCleaner, updateInterval, err := loadConfig()
	if err != nil {
		gui.PrintError("Failed to load config: %v", err)
		os.Exit(1)
	}

	gui.PrintConfig(autoCleaner.CleanAbove, autoCleaner.CleanInterval)
	gui.PrintMonitor("Monitoring memory usage. Press Enter to clean RAM manually. Press Ctrl+C to exit...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		gui.PrintInfo("Exiting...")
		cancel()
	}()

	inputChan := make(chan struct{})
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			_, err := reader.ReadBytes('\n')
			if err != nil {
				break
			}
			select {
			case inputChan <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
	}()

	var lastClean time.Time

	var autoCleanTicker *time.Ticker
	var autoCleanC <-chan time.Time
	if autoCleaner.CleanInterval > 0 {
		autoCleanTicker = time.NewTicker(time.Duration(autoCleaner.CleanInterval) * time.Minute)
		autoCleanC = autoCleanTicker.C
		defer autoCleanTicker.Stop()
	}

	monitorC := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				monitorC <- struct{}{}
				time.Sleep(time.Duration(updateInterval * float64(time.Second)))
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-inputChan:
			gui.PrintManualCleanStart()

			if err := memory.CleanRam(); err != nil {
				gui.PrintError("Failed to clean RAM: %v", err)
			} else {
				gui.PrintManualCleanComplete()
			}
			lastClean = time.Now()

		case <-monitorC:
			stats, err := monitor.GetRamStats()
			if err != nil {
				gui.PrintError("Failed to get memory stats: %v", err)
				continue
			}

			gui.PrintMemoryStats(stats.UsedPercent, stats.AvailableGB, stats.TotalGB, autoCleaner.CleanAbove)

			if stats.UsedPercent >= float64(autoCleaner.CleanAbove) &&
				time.Since(lastClean) > 30*time.Second {
				gui.PrintCleaningStart(float64(autoCleaner.CleanAbove), stats.UsedPercent)

				if err := memory.CleanRam(); err != nil {
					gui.PrintError("Failed to clean RAM: %v", err)
				} else {
					gui.PrintCleaningComplete()
				}
				lastClean = time.Now()
			}

		case <-autoCleanC:
			if time.Since(lastClean) > time.Minute {
				gui.PrintAuto("Scheduled cleaning every %d minutes.", autoCleaner.CleanInterval)

				if err := memory.CleanRam(); err != nil {
					gui.PrintError("Failed to clean RAM: %v", err)
				} else {
					gui.PrintCleaned("RAM cleaned by interval.")
				}
				lastClean = time.Now()
			}
		}
	}
}
