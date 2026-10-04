package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"lowkey/pkg/engine"
	"lowkey/pkg/osutil"
	"lowkey/pkg/profile"
	"lowkey/pkg/router"
	"lowkey/pkg/ui"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingTop(0).
			PaddingBottom(0).
			PaddingLeft(2).
			PaddingRight(2)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)
)

func main() {
	throttler := osutil.NewOSThrottler()
	defer throttler.Cleanup()

	// Initial choice: Load saved setup OR Create new setup
	savedProfiles, _ := profile.ListProfiles()
	availableEngines := engine.DetectAvailable()

	// Pretty Splash Screen
	fmt.Print(ui.RenderSplash(len(availableEngines), len(savedProfiles), throttler.IsOnBattery()))

	var startAction string
	var startOptions []huh.Option[string]

	if len(savedProfiles) > 0 {
		startOptions = append(startOptions, huh.NewOption(fmt.Sprintf("📂 Load Saved Setup (%d available)", len(savedProfiles)), "load"))
	}
	startOptions = append(startOptions,
		huh.NewOption("✨ Create New Setup (Interactive Wizard)", "new"),
		huh.NewOption("🚀 Start Multi-Model Router", "router"),
		huh.NewOption("⚙️  Create Router Config File", "router-config"),
		huh.NewOption("❌ Exit", "exit"),
	)

	err := huh.NewSelect[string]().
		Title("How would you like to start?").
		Options(startOptions...).
		Value(&startAction).
		Run()

	if err != nil || startAction == "exit" {
		fmt.Println("Aborted.")
		return
	}

	if startAction == "router" {
		startRouterMode()
		return
	}

	if startAction == "router-config" {
		createRouterConfig()
		return
	}

	var activeConfig engine.LaunchConfig
	var selectedEngine engine.Engine

	if startAction == "load" {
		var profileName string
		var profileOptions []huh.Option[string]
		for _, p := range savedProfiles {
			desc := fmt.Sprintf("%s (%s | %s)", p.Name, p.Config.EngineID, p.Config.ThermalProfile)
			profileOptions = append(profileOptions, huh.NewOption(desc, p.Name))
		}

		err = huh.NewSelect[string]().
			Title("Choose a saved setup to load:").
			Options(profileOptions...).
			Value(&profileName).
			Run()
		if err != nil {
			fmt.Println("Aborted.")
			return
		}

		loadedProfile, err := profile.LoadProfile(profileName)
		if err != nil {
			fmt.Println(warnStyle.Render(fmt.Sprintf("Failed to load profile: %v", err)))
			return
		}

		activeConfig = loadedProfile.Config
		selectedEngine, err = engine.Get(activeConfig.EngineID)
		if err != nil {
			fmt.Println(warnStyle.Render(fmt.Sprintf("Configured engine '%s' is not registered: %v", activeConfig.EngineID, err)))
			return
		}

		fmt.Println(infoStyle.Render(fmt.Sprintf("✔ Loaded setup '%s' [%s]", loadedProfile.Name, selectedEngine.Name())))
	} else {
		// New setup wizard
		cfg, eng, err := runNewSetupWizard()
		if err != nil {
			fmt.Println("Wizard canceled:", err)
			return
		}
		activeConfig = *cfg
		selectedEngine = eng

		// Ask if user wants to save this setup
		var wantSave bool
		_ = huh.NewConfirm().
			Title("Would you like to save this setup for future one-click launches?").
			Value(&wantSave).
			Run()

		if wantSave {
			var profileName string
			_ = huh.NewInput().
				Title("Setup Profile Name:").
				Placeholder("e.g. daily-coding, quiet-agent, heavy-throughput").
				Value(&profileName).
				WithWidth(60).
				Run()

			if strings.TrimSpace(profileName) != "" {
				p := profile.Profile{
					Name:   strings.TrimSpace(profileName),
					Config: activeConfig,
				}
				savedPath, err := profile.SaveProfile(p)
				if err != nil {
					fmt.Println(warnStyle.Render(fmt.Sprintf("Failed to save profile: %v", err)))
				} else {
					fmt.Println(infoStyle.Render(fmt.Sprintf("✔ Saved profile to: %s", savedPath)))
				}
			}
		}
	}

	// Launch engine with Throttler
	launchEngine(throttler, selectedEngine, &activeConfig)
}

func runNewSetupWizard() (*engine.LaunchConfig, engine.Engine, error) {
	// 1. Detect engines
	available := engine.DetectAvailable()
	all := engine.GetAll()

	var engineOptions []huh.Option[string]
	if len(available) > 0 {
		for _, eng := range available {
			_, binPath := eng.IsInstalled()
			label := fmt.Sprintf("✅ %s (Found at %s)", eng.Name(), binPath)
			engineOptions = append(engineOptions, huh.NewOption(label, eng.ID()))
		}
	}

	// Add uninstalled engines as well with warning marker
	for _, eng := range all {
		if installed, _ := eng.IsInstalled(); !installed {
			label := fmt.Sprintf("⚠️  %s (Not detected in PATH)", eng.Name())
			engineOptions = append(engineOptions, huh.NewOption(label, eng.ID()))
		}
	}

	var selectedEngineID string
	err := huh.NewSelect[string]().
		Title("Select Inference Engine:").
		Description("Green checkmarks are auto-detected on your system").
		Options(engineOptions...).
		Value(&selectedEngineID).
		Run()
	if err != nil {
		return nil, nil, err
	}

	selectedEngine, _ := engine.Get(selectedEngineID)

	// 2. Discover models for this engine
	models, _ := selectedEngine.DiscoverModels()
	var modelOptions []huh.Option[string]
	for _, m := range models {
		modelOptions = append(modelOptions, huh.NewOption(m.DisplayName, m.Path))
	}
	modelOptions = append(modelOptions, huh.NewOption("➕ Custom Model Path / ID...", "custom"))

	var selectedModel string
	err = huh.NewSelect[string]().
		Title(fmt.Sprintf("Select Model for %s:", selectedEngine.Name())).
		Description(fmt.Sprintf("Discovered %d models in standard directories", len(models))).
		Options(modelOptions...).
		Value(&selectedModel).
		Run()
	if err != nil {
		return nil, nil, err
	}

	if selectedModel == "custom" {
		err = huh.NewInput().
			Title("Enter custom Model Path or Identifier:").
			Placeholder("/path/to/model or huggingface/repo").
			Value(&selectedModel).
			Run()
		if err != nil {
			return nil, nil, err
		}
	}

	// 3. Port & Host configuration
	portStr := strconv.Itoa(selectedEngine.DefaultPort())
	hostStr := "127.0.0.1"

	// 4. Engine-specific options
	var preset string = "custom"
	var thinkingEffort string = "medium"
	var speculationDepthStr string = "3"
	var contextSizeStr string = "8192"

	formFields := []huh.Field{
		huh.NewInput().
			Title("Server Port:").
			Value(&portStr),
		huh.NewInput().
			Title("Host Bind Address:").
			Value(&hostStr),
	}

	if selectedEngine.ID() == "mtplx" {
		formFields = append(formFields,
			huh.NewSelect[string]().
				Title("Throughput Preset:").
				Options(
					huh.NewOption("throughput-4 (Recommended for Speculation)", "throughput-4"),
					huh.NewOption("long-context (Agent / Repositories)", "long-context"),
					huh.NewOption("throughput-8 (Extreme)", "throughput-8"),
					huh.NewOption("custom", "custom"),
				).
				Value(&preset),
			huh.NewSelect[string]().
				Title("Thinking / Reasoning Effort:").
				Options(
					huh.NewOption("low (Fast answers)", "low"),
					huh.NewOption("medium (Balanced)", "medium"),
					huh.NewOption("high (Deep reasoning / coding)", "high"),
					huh.NewOption("off (No thinking tokens)", "off"),
				).
				Value(&thinkingEffort),
			huh.NewInput().
				Title("Speculation Depth:").
				Value(&speculationDepthStr),
		)
	} else if selectedEngine.ID() == "llamacpp" || selectedEngine.ID() == "lmstudio" {
		formFields = append(formFields,
			huh.NewInput().
				Title("Context Window Size (Tokens):").
				Value(&contextSizeStr),
		)
		speculationDepthStr = "0"
		if selectedEngine.ID() == "llamacpp" && engine.IsMTPModel(selectedModel) {
			speculationDepthStr = "2"
			formFields = append(formFields,
				huh.NewInput().
					Title("MTP Speculation Depth (draft tokens, 0 = off):").
					Value(&speculationDepthStr),
			)
		}
	}

	err = huh.NewForm(huh.NewGroup(formFields...)).Run()
	if err != nil {
		return nil, nil, err
	}

	// 5. Thermal & Power Throttling Profile
	var selectedThermal string
	var thermalOptions []huh.Option[string]
	for _, p := range osutil.GetAllThermalProfiles() {
		thermalOptions = append(thermalOptions, huh.NewOption(p.Description, p.Name))
	}

	err = huh.NewSelect[string]().
		Title("Thermal & Power Throttling Profile:").
		Description("Controls background duty-cycle pausing and OS scheduling priority").
		Options(thermalOptions...).
		Value(&selectedThermal).
		Run()
	if err != nil {
		return nil, nil, err
	}

	portNum, _ := strconv.Atoi(portStr)
	ctxNum, _ := strconv.Atoi(contextSizeStr)
	specNum, _ := strconv.Atoi(speculationDepthStr)

	cfg := &engine.LaunchConfig{
		EngineID:         selectedEngine.ID(),
		ModelID:          selectedModel,
		ModelPath:        selectedModel,
		Port:             portNum,
		Host:             hostStr,
		Preset:           preset,
		ThinkingEffort:   thinkingEffort,
		SpeculationDepth: specNum,
		ContextSize:      ctxNum,
		ThermalProfile:   selectedThermal,
	}

	return cfg, selectedEngine, nil
}

func startRouterMode() {
	// Ask for config file path
	var configPath string
	_ = huh.NewInput().
		Title("Router Config File Path:").
		Placeholder("/path/to/router.json").
		Value(&configPath).
		Run()

	if strings.TrimSpace(configPath) == "" {
		fmt.Println("Aborted: no config file path provided.")
		return
	}

	cfg, err := router.LoadRouterConfig(configPath)
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("Failed to load router config: %v", err)))
		return
	}

	r, err := router.NewRouter(cfg)
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("Failed to create router: %v", err)))
		return
	}

	fmt.Println(infoStyle.Render("🚀 Starting Multi-Model Router..."))
	fmt.Printf("   Host:        %s\n", cfg.Host)
	fmt.Printf("   Port:        %d\n", cfg.Port)
	fmt.Printf("   Models:      %d\n", len(cfg.Models))
	fmt.Printf("   Idle Timeout: %d minutes\n", cfg.IdleTimeoutMin)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n🛑 Shutting down router...")
		r.Shutdown()
	}()

	if err := r.Run(); err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("Router error: %v", err)))
		return
	}
}

func createRouterConfig() {
	fmt.Println(titleStyle.Render(" Router Config Creator "))
	fmt.Println("Create a router config file that defines multiple models with one API endpoint.")
	fmt.Println()

	// Base router settings
	var host string = "127.0.0.1"
	var portStr string = "8000"
	var idleTimeoutStr string = "10"
	var memoryReserveStr string = "10"

	_ = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Host (leave blank for localhost)").Value(&host),
		huh.NewInput().Title("Port").Value(&portStr),
		huh.NewInput().Title("Idle timeout (minutes)").Value(&idleTimeoutStr),
		huh.NewInput().Title("Memory reserve (% for OS)").Value(&memoryReserveStr),
	)).Run()

	if strings.TrimSpace(host) == "" {
		host = "127.0.0.1"
	}

	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 8000
	}
	idleTimeout, _ := strconv.Atoi(idleTimeoutStr)
	if idleTimeout == 0 {
		idleTimeout = 10
	}
	memReserve, _ := strconv.Atoi(memoryReserveStr)

	// Model configuration loop
	models := make(map[string]*router.ModelConfig)

	for {
		var modelName string
		_ = huh.NewInput().
			Title("Model name (API identifier, e.g. gpt-4o-mini)").
			Value(&modelName).
			Run()
		modelName = strings.TrimSpace(modelName)

		if modelName == "" {
			break
		}

		// Engine selection
		available := engine.DetectAvailable()
		var engineOptions []huh.Option[string]
		for _, eng := range available {
			_, binPath := eng.IsInstalled()
			label := fmt.Sprintf("✅ %s (%s)", eng.Name(), binPath)
			engineOptions = append(engineOptions, huh.NewOption(label, eng.ID()))
		}

		var selectedEngineID string
		_ = huh.NewSelect[string]().
			Title("Engine for this model").
			Options(engineOptions...).
			Value(&selectedEngineID).
			Run()

		eng, _ := engine.Get(selectedEngineID)

		// Model path
		var modelPath string
		modelsList, _ := eng.DiscoverModels()
		if len(modelsList) > 0 {
			var modelOptions []huh.Option[string]
			for _, m := range modelsList {
				modelOptions = append(modelOptions, huh.NewOption(m.DisplayName, m.Path))
			}
			modelOptions = append(modelOptions, huh.NewOption("➕ Custom path", "custom"))

			_ = huh.NewSelect[string]().
				Title("Model path").
				Options(modelOptions...).
				Value(&modelPath).
				Run()

			if modelPath == "custom" {
				_ = huh.NewInput().
					Title("Enter model path").
					Value(&modelPath).
					Run()
			}
		} else {
			_ = huh.NewInput().
				Title("Model path").
				Value(&modelPath).
				Run()
		}

		// Thermal profile
		var thermalProfile string = "quiet"
		var thermalOptions []huh.Option[string]
		for _, p := range osutil.GetAllThermalProfiles() {
			thermalOptions = append(thermalOptions, huh.NewOption(p.Description, p.Name))
		}
		_ = huh.NewSelect[string]().
			Title("Thermal profile").
			Options(thermalOptions...).
			Value(&thermalProfile).
			Run()

		models[modelName] = &router.ModelConfig{
			Engine:         selectedEngineID,
			Path:           modelPath,
			ContextSize:    8192,
			ThermalProfile: thermalProfile,
		}

		fmt.Println(infoStyle.Render(fmt.Sprintf("✔ Added model '%s' -> %s", modelName, selectedEngineID)))

		var addAnother bool
		_ = huh.NewConfirm().
			Title("Add another model?").
			Value(&addAnother).
			Run()
		if !addAnother {
			break
		}
	}

	if len(models) == 0 {
		fmt.Println(warnStyle.Render("No models added. Router config not created."))
		return
	}

	// Save config
	var savePath string = "~/.lowkey/router.json"
	_ = huh.NewInput().
		Title("Save config to").
		Value(&savePath).
		Run()

	if strings.HasPrefix(savePath, "~/") {
		home, _ := os.UserHomeDir()
		savePath = filepath.Join(home, savePath[2:])
	}

	cfg := router.RouterConfig{
		Host:             host,
		Port:             port,
		MemoryReservePct: memReserve,
		IdleTimeoutMin:   idleTimeout,
		Models:           models,
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("Failed to serialize config: %v", err)))
		return
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("Failed to write config: %v", err)))
		return
	}

	fmt.Println(infoStyle.Render(fmt.Sprintf("✔ Router config saved to: %s", savePath)))
	fmt.Println(subtleStyle.Render("Run lowkey and choose 'Start Multi-Model Router' to use it."))
}

func launchEngine(throttler osutil.OSThrottler, eng engine.Engine, cfg *engine.LaunchConfig) {
	fmt.Println()
	fmt.Println(infoStyle.Render("🚀 Starting Inference Engine..."))
	fmt.Printf("   Engine:  %s\n", eng.Name())
	fmt.Printf("   Model:   %s\n", cfg.ModelPath)
	fmt.Printf("   Port:    %d\n", cfg.Port)
	fmt.Printf("   Thermal: %s\n", cfg.ThermalProfile)

	if throttler.IsOnBattery() {
		fmt.Println(warnStyle.Render("   🔋 Battery power detected! Throttling will extend battery life."))
	} else {
		fmt.Println(subtleStyle.Render("   🔌 AC / Wall power connected."))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd, err := eng.BuildCommand(ctx, cfg)
	if err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("Failed to build command: %v", err)))
		return
	}

	throttler.ConfigureCommand(cmd)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Println(warnStyle.Render(fmt.Sprintf("Failed to launch process: %v", err)))
		return
	}

	pid := cmd.Process.Pid
	fmt.Println(infoStyle.Render(fmt.Sprintf("✔ Process started with PID: %d", pid)))

	// Apply OS priority & background QoS
	if err := throttler.OnProcessStarted(pid); err != nil {
		fmt.Println(subtleStyle.Render(fmt.Sprintf("Notice: OS priority adjustment: %v", err)))
	}

	// Match thermal profile
	var activeProfile osutil.ThermalProfile = osutil.ProfileQuiet
	for _, p := range osutil.GetAllThermalProfiles() {
		if p.Name == cfg.ThermalProfile {
			activeProfile = p
			break
		}
	}

	// Run duty cycle loop in background
	if activeProfile.WorkTime > 0 && activeProfile.PauseTime > 0 {
		fmt.Println(infoStyle.Render(fmt.Sprintf("⚡ Duty-cycle throttling active: %v work / %v pause", activeProfile.WorkTime, activeProfile.PauseTime)))
		go throttler.StartDutyCycleLoop(ctx, pid, activeProfile)
	} else {
		fmt.Println(subtleStyle.Render("⚡ Running at 100% duty cycle (priority scheduling only)"))
	}

	// Clean shutdown handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-sigChan
		fmt.Println("\n🛑 Caught exit signal, shutting down gracefully...")
		cancel()
		_ = throttler.ResumeProcess(pid)
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}()

	err = cmd.Wait()
	throttler.Cleanup()

	if err != nil && !strings.Contains(err.Error(), "signal: terminated") && !strings.Contains(err.Error(), "signal: interrupt") {
		fmt.Println(warnStyle.Render(fmt.Sprintf("Process exited with code: %v", err)))
	} else {
		fmt.Println(infoStyle.Render("Server stopped cleanly."))
	}
}
