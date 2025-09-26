package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// LspInstaller handles installation and version management of language servers
// Inspired by Zed's LspInstaller trait
type LspInstaller interface {
	// BinaryVersion represents a version of the language server binary
	BinaryVersion() string
	
	// CheckIfUserInstalled checks if user has manually installed the server
	CheckIfUserInstalled(delegate LanguageServerDelegate) (*LanguageServerBinary, error)
	
	// FetchLatestServerVersion gets the latest available version
	FetchLatestServerVersion(ctx context.Context, delegate LanguageServerDelegate) (string, error)
	
	// CheckIfVersionInstalled checks if a specific version is installed locally
	CheckIfVersionInstalled(version string, containerDir string, delegate LanguageServerDelegate) (*LanguageServerBinary, error)
	
	// FetchServerBinary downloads and installs a specific version
	FetchServerBinary(ctx context.Context, version string, containerDir string, delegate LanguageServerDelegate) (*LanguageServerBinary, error)
	
	// CachedServerBinary returns the cached server binary if available
	CachedServerBinary(containerDir string, delegate LanguageServerDelegate) (*LanguageServerBinary, error)
	
	// GetInstallationInfo returns information about installation requirements
	GetInstallationInfo() InstallationInfo
}

// InstallationInfo contains information about how to install a language server
type InstallationInfo struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	SupportedOS     []string `json:"supported_os"`
	RequiredTools   []string `json:"required_tools"`
	InstallMethods  []string `json:"install_methods"`
	DefaultMethod   string   `json:"default_method"`
}

// TypeScriptLspInstaller implements LspInstaller for TypeScript language servers
type TypeScriptLspInstaller struct {
	serverType ServerType
	version    string
}

// NewTypeScriptLspInstaller creates a new TypeScript LSP installer
func NewTypeScriptLspInstaller(serverType ServerType) *TypeScriptLspInstaller {
	return &TypeScriptLspInstaller{
		serverType: serverType,
	}
}

// BinaryVersion implements LspInstaller.BinaryVersion
func (i *TypeScriptLspInstaller) BinaryVersion() string {
	return i.version
}

// CheckIfUserInstalled implements LspInstaller.CheckIfUserInstalled
func (i *TypeScriptLspInstaller) CheckIfUserInstalled(delegate LanguageServerDelegate) (*LanguageServerBinary, error) {
	var command string
	var args []string
	
	switch i.serverType {
	case ServerTypeVTSLS:
		command = "vtsls"
		args = []string{"--stdio"}
	case ServerTypeTypeScriptLanguageServer:
		command = "typescript-language-server"
		args = []string{"--stdio"}
	default:
		return nil, fmt.Errorf("unsupported server type")
	}
	
	// Check if command exists in PATH
	path, err := delegate.Which(command)
	if err != nil {
		return nil, err
	}
	
	return &LanguageServerBinary{
		Path: path,
		Args: args,
		Env:  delegate.ShellEnv(),
	}, nil
}

// FetchLatestServerVersion implements LspInstaller.FetchLatestServerVersion
func (i *TypeScriptLspInstaller) FetchLatestServerVersion(ctx context.Context, delegate LanguageServerDelegate) (string, error) {
	var packageName string
	switch i.serverType {
	case ServerTypeVTSLS:
		packageName = "@vtsls/language-server"
	case ServerTypeTypeScriptLanguageServer:
		packageName = "typescript-language-server"
	default:
		return "", fmt.Errorf("unsupported server type")
	}
	
	// Check npm registry for latest version
	url := fmt.Sprintf("https://registry.npmjs.org/%s/latest", packageName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch package info: %s", resp.Status)
	}
	
	var packageInfo struct {
		Version string `json:"version"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&packageInfo); err != nil {
		return "", err
	}
	
	i.version = packageInfo.Version
	return packageInfo.Version, nil
}

// CheckIfVersionInstalled implements LspInstaller.CheckIfVersionInstalled
func (i *TypeScriptLspInstaller) CheckIfVersionInstalled(version string, containerDir string, delegate LanguageServerDelegate) (*LanguageServerBinary, error) {
	serverName := i.getServerName()
	versionDir := filepath.Join(containerDir, serverName, version)
	
	// Check if version directory exists
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return nil, nil
	}
	
	// Check for binary
	binaryPath := i.getBinaryPath(versionDir)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, nil
	}
	
	return &LanguageServerBinary{
		Path: binaryPath,
		Args: i.getBinaryArgs(),
		Env:  delegate.ShellEnv(),
	}, nil
}

// FetchServerBinary implements LspInstaller.FetchServerBinary
func (i *TypeScriptLspInstaller) FetchServerBinary(ctx context.Context, version string, containerDir string, delegate LanguageServerDelegate) (*LanguageServerBinary, error) {
	serverName := i.getServerName()
	versionDir := filepath.Join(containerDir, serverName, version)
	
	// Create version directory
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create version directory: %w", err)
	}
	
	// Install using npm into specific directory
	if err := i.installToDirectory(ctx, versionDir, version); err != nil {
		return nil, fmt.Errorf("failed to install server: %w", err)
	}
	
	// Return binary info
	binaryPath := i.getBinaryPath(versionDir)
	return &LanguageServerBinary{
		Path: binaryPath,
		Args: i.getBinaryArgs(),
		Env:  delegate.ShellEnv(),
	}, nil
}

// CachedServerBinary implements LspInstaller.CachedServerBinary
func (i *TypeScriptLspInstaller) CachedServerBinary(containerDir string, delegate LanguageServerDelegate) (*LanguageServerBinary, error) {
	serverName := i.getServerName()
	serverDir := filepath.Join(containerDir, serverName)
	
	// Find the latest installed version
	entries, err := os.ReadDir(serverDir)
	if err != nil {
		return nil, nil // No cached versions
	}
	
	var latestVersion string
	for _, entry := range entries {
		if entry.IsDir() {
			// Simple version comparison (could be improved with proper semver)
			if latestVersion == "" || entry.Name() > latestVersion {
				latestVersion = entry.Name()
			}
		}
	}
	
	if latestVersion == "" {
		return nil, nil
	}
	
	return i.CheckIfVersionInstalled(latestVersion, containerDir, delegate)
}

// GetInstallationInfo implements LspInstaller.GetInstallationInfo
func (i *TypeScriptLspInstaller) GetInstallationInfo() InstallationInfo {
	serverName := i.getServerName()
	
	return InstallationInfo{
		Name:        serverName,
		Description: i.getDescription(),
		SupportedOS: []string{"linux", "darwin", "windows"},
		RequiredTools: []string{"node", "npm"},
		InstallMethods: []string{"npm", "local_directory"},
		DefaultMethod: "local_directory",
	}
}

// Helper methods

func (i *TypeScriptLspInstaller) getServerName() string {
	switch i.serverType {
	case ServerTypeVTSLS:
		return "vtsls"
	case ServerTypeTypeScriptLanguageServer:
		return "typescript-language-server"
	default:
		return "unknown"
	}
}

func (i *TypeScriptLspInstaller) getDescription() string {
	switch i.serverType {
	case ServerTypeVTSLS:
		return "Vue TypeScript Language Server - Advanced TypeScript/JavaScript language server"
	case ServerTypeTypeScriptLanguageServer:
		return "TypeScript Language Server - Standard TypeScript/JavaScript language server"
	default:
		return "Unknown TypeScript language server"
	}
}

func (i *TypeScriptLspInstaller) getBinaryPath(installDir string) string {
	serverName := i.getServerName()
	
	switch i.serverType {
	case ServerTypeVTSLS:
		if runtime.GOOS == "windows" {
			return filepath.Join(installDir, "node_modules", ".bin", "vtsls.cmd")
		}
		return filepath.Join(installDir, "node_modules", ".bin", "vtsls")
	case ServerTypeTypeScriptLanguageServer:
		if runtime.GOOS == "windows" {
			return filepath.Join(installDir, "node_modules", ".bin", "typescript-language-server.cmd")
		}
		return filepath.Join(installDir, "node_modules", ".bin", "typescript-language-server")
	default:
		return filepath.Join(installDir, serverName)
	}
}

func (i *TypeScriptLspInstaller) getBinaryArgs() []string {
	return []string{"--stdio"}
}

func (i *TypeScriptLspInstaller) installToDirectory(ctx context.Context, installDir string, version string) error {
	var packageName string
	switch i.serverType {
	case ServerTypeVTSLS:
		if version != "" {
			packageName = fmt.Sprintf("@vtsls/language-server@%s", version)
		} else {
			packageName = "@vtsls/language-server"
		}
	case ServerTypeTypeScriptLanguageServer:
		if version != "" {
			packageName = fmt.Sprintf("typescript-language-server@%s", version)
		} else {
			packageName = "typescript-language-server"
		}
		// Also install TypeScript as dependency
		defer func() {
			cmd := exec.CommandContext(ctx, "npm", "install", "typescript", "--prefix", installDir)
			cmd.Run() // Ignore errors for TypeScript installation
		}()
	default:
		return fmt.Errorf("unsupported server type")
	}
	
	// Install package to specific directory
	cmd := exec.CommandContext(ctx, "npm", "install", packageName, "--prefix", ".")
	cmd.Dir = installDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install failed: %w\nOutput: %s", err, string(output))
	}
	
	return nil
}

// InstallationManager manages language server installations across different directories
type InstallationManager struct {
	baseDir   string
	installers map[string]LspInstaller
}

// NewInstallationManager creates a new installation manager
func NewInstallationManager(baseDir string) *InstallationManager {
	if baseDir == "" {
		// Default to user's cache directory
		homeDir, _ := os.UserHomeDir()
		baseDir = filepath.Join(homeDir, ".cache", "ts-index", "lsp-servers")
	}
	
	manager := &InstallationManager{
		baseDir:    baseDir,
		installers: make(map[string]LspInstaller),
	}
	
	// Register built-in installers
	manager.RegisterInstaller("vtsls", NewTypeScriptLspInstaller(ServerTypeVTSLS))
	manager.RegisterInstaller("typescript-language-server", NewTypeScriptLspInstaller(ServerTypeTypeScriptLanguageServer))
	
	return manager
}

// RegisterInstaller registers a new installer
func (m *InstallationManager) RegisterInstaller(name string, installer LspInstaller) {
	m.installers[name] = installer
}

// InstallServer installs a language server
func (m *InstallationManager) InstallServer(ctx context.Context, serverName string, version string, delegate LanguageServerDelegate) (*LanguageServerBinary, error) {
	installer, exists := m.installers[serverName]
	if !exists {
		return nil, fmt.Errorf("no installer found for server: %s", serverName)
	}
	
	// Create base directory
	if err := os.MkdirAll(m.baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	
	// If no version specified, fetch latest
	if version == "" {
		var err error
		version, err = installer.FetchLatestServerVersion(ctx, delegate)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch latest version: %w", err)
		}
	}
	
	// Check if already installed
	if binary, err := installer.CheckIfVersionInstalled(version, m.baseDir, delegate); err == nil && binary != nil {
		return binary, nil
	}
	
	// Install the server
	return installer.FetchServerBinary(ctx, version, m.baseDir, delegate)
}

// GetInstalledServers returns information about installed servers
func (m *InstallationManager) GetInstalledServers(delegate LanguageServerDelegate) ([]InstalledServerInfo, error) {
	var servers []InstalledServerInfo
	
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return servers, nil // No installations yet
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		serverName := entry.Name()
		serverDir := filepath.Join(m.baseDir, serverName)
		
		// Get versions
		versionEntries, err := os.ReadDir(serverDir)
		if err != nil {
			continue
		}
		
		var versions []string
		for _, vEntry := range versionEntries {
			if vEntry.IsDir() {
				versions = append(versions, vEntry.Name())
			}
		}
		
		if len(versions) > 0 {
			servers = append(servers, InstalledServerInfo{
				Name:     serverName,
				Versions: versions,
				Path:     serverDir,
			})
		}
	}
	
	return servers, nil
}

// GetServerBinary gets the binary for a specific server (latest version if not specified)
func (m *InstallationManager) GetServerBinary(serverName string, version string, delegate LanguageServerDelegate) (*LanguageServerBinary, error) {
	installer, exists := m.installers[serverName]
	if !exists {
		return nil, fmt.Errorf("no installer found for server: %s", serverName)
	}
	
	if version == "" {
		// Get cached (latest) version
		return installer.CachedServerBinary(m.baseDir, delegate)
	}
	
	// Get specific version
	return installer.CheckIfVersionInstalled(version, m.baseDir, delegate)
}

// CleanupServer removes old versions of a server, keeping only the latest N versions
func (m *InstallationManager) CleanupServer(serverName string, keepVersions int) error {
	serverDir := filepath.Join(m.baseDir, serverName)
	
	entries, err := os.ReadDir(serverDir)
	if err != nil {
		return nil // Nothing to clean
	}
	
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}
	
	if len(versions) <= keepVersions {
		return nil // Nothing to clean
	}
	
	// Sort versions (simple string sort, could be improved with semver)
	// Keep the latest N versions, remove the rest
	for i := 0; i < len(versions)-keepVersions; i++ {
		versionDir := filepath.Join(serverDir, versions[i])
		os.RemoveAll(versionDir)
	}
	
	return nil
}

// InstalledServerInfo contains information about an installed server
type InstalledServerInfo struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
	Path     string   `json:"path"`
}