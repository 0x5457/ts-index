package lsp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// LanguageServerManager manages multiple language servers across different workspaces
// This follows Zed's pattern of managing language servers per workspace
type LanguageServerManager struct {
	adapters map[string]LspAdapter      // language name -> adapter
	servers  map[string]*LanguageServer // workspace_root:language -> server
	delegate LanguageServerDelegate
	mu       sync.RWMutex
}

// NewLanguageServerManager creates a new language server manager
func NewLanguageServerManager(delegate LanguageServerDelegate) *LanguageServerManager {
	manager := &LanguageServerManager{
		adapters: make(map[string]LspAdapter),
		servers:  make(map[string]*LanguageServer),
		delegate: delegate,
	}
	
	// Register built-in adapters
	manager.RegisterAdapter("typescript", NewTypeScriptLspAdapter())
	manager.RegisterAdapter("javascript", NewTypeScriptLspAdapter())
	manager.RegisterAdapter("typescriptreact", NewTypeScriptLspAdapter())
	manager.RegisterAdapter("javascriptreact", NewTypeScriptLspAdapter())
	
	return manager
}

// RegisterAdapter registers a language adapter
func (m *LanguageServerManager) RegisterAdapter(language string, adapter LspAdapter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.adapters[language] = adapter
}

// GetLanguageServer gets or creates a language server for the given workspace and language
func (m *LanguageServerManager) GetLanguageServer(ctx context.Context, workspaceRoot, language string) (*LanguageServer, error) {
	key := m.serverKey(workspaceRoot, language)
	
	m.mu.RLock()
	if server, exists := m.servers[key]; exists && server.IsRunning() {
		m.mu.RUnlock()
		return server, nil
	}
	m.mu.RUnlock()
	
	// Need to create a new server
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double-check after acquiring write lock
	if server, exists := m.servers[key]; exists && server.IsRunning() {
		return server, nil
	}
	
	// Get adapter for this language
	adapter, exists := m.adapters[language]
	if !exists {
		return nil, fmt.Errorf("no adapter registered for language: %s", language)
	}
	
	// Check if the adapter's language server is installed
	if !adapter.IsInstalled() {
		return nil, fmt.Errorf("language server for %s is not installed. Adapter: %s", language, adapter.Name())
	}
	
	// Create absolute workspace path
	absWorkspace, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute workspace path: %w", err)
	}
	
	// Create new language server
	server := NewLanguageServer(adapter, m.delegate, absWorkspace)
	
	// Start the server
	if err := server.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start language server: %w", err)
	}
	
	// Store the server
	m.servers[key] = server
	
	return server, nil
}

// StopLanguageServer stops a language server for a specific workspace and language
func (m *LanguageServerManager) StopLanguageServer(workspaceRoot, language string) error {
	key := m.serverKey(workspaceRoot, language)
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if server, exists := m.servers[key]; exists {
		err := server.Stop()
		delete(m.servers, key)
		return err
	}
	
	return nil
}

// StopWorkspaceServers stops all language servers for a specific workspace
func (m *LanguageServerManager) StopWorkspaceServers(workspaceRoot string) error {
	absWorkspace, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return err
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var lastErr error
	for key, server := range m.servers {
		if m.matchesWorkspace(key, absWorkspace) {
			if err := server.Stop(); err != nil {
				lastErr = err
			}
			delete(m.servers, key)
		}
	}
	
	return lastErr
}

// StopAllServers stops all language servers
func (m *LanguageServerManager) StopAllServers() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var lastErr error
	for key, server := range m.servers {
		if err := server.Stop(); err != nil {
			lastErr = err
		}
		delete(m.servers, key)
	}
	
	return lastErr
}

// GetRunningServers returns information about running servers
func (m *LanguageServerManager) GetRunningServers() []ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var servers []ServerInfo
	for key, server := range m.servers {
		if server.IsRunning() {
			servers = append(servers, ServerInfo{
				Key:           key,
				Name:          server.Name(),
				WorkspaceRoot: server.RootPath(),
				AdapterName:   server.Adapter().Name(),
			})
		}
	}
	
	return servers
}

// GetRegisteredAdapters returns information about registered adapters
func (m *LanguageServerManager) GetRegisteredAdapters() []AdapterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var adapters []AdapterInfo
	for language, adapter := range m.adapters {
		adapters = append(adapters, AdapterInfo{
			Language:    language,
			Name:        adapter.Name(),
			IsInstalled: adapter.IsInstalled(),
			CanInstall:  adapter.CanInstall(),
		})
	}
	
	return adapters
}

// InstallLanguageServer installs a language server for the given language
func (m *LanguageServerManager) InstallLanguageServer(ctx context.Context, language string) error {
	m.mu.RLock()
	adapter, exists := m.adapters[language]
	m.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("no adapter registered for language: %s", language)
	}
	
	if !adapter.CanInstall() {
		return fmt.Errorf("adapter for %s cannot install language server", language)
	}
	
	return adapter.Install(ctx)
}

// Helper functions

func (m *LanguageServerManager) serverKey(workspaceRoot, language string) string {
	absWorkspace, _ := filepath.Abs(workspaceRoot)
	return fmt.Sprintf("%s:%s", absWorkspace, language)
}

func (m *LanguageServerManager) matchesWorkspace(key, workspaceRoot string) bool {
	// Key format is "workspace:language"
	if len(key) > len(workspaceRoot) && key[:len(workspaceRoot)] == workspaceRoot {
		// Check if it's followed by a colon (not just a prefix match)
		return len(key) > len(workspaceRoot) && key[len(workspaceRoot)] == ':'
	}
	return false
}

// Information types

type ServerInfo struct {
	Key           string
	Name          string
	WorkspaceRoot string
	AdapterName   string
}

type AdapterInfo struct {
	Language    string
	Name        string
	IsInstalled bool
	CanInstall  bool
}

// DefaultDelegate provides a basic implementation of LanguageServerDelegate
type DefaultDelegate struct {
	workspaceRoot string
}

// NewDefaultDelegate creates a new default delegate
func NewDefaultDelegate(workspaceRoot string) *DefaultDelegate {
	return &DefaultDelegate{workspaceRoot: workspaceRoot}
}

// ReadTextFile implements LanguageServerDelegate.ReadTextFile
func (d *DefaultDelegate) ReadTextFile(path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(d.workspaceRoot, path)
	}
	
	content, err := readFileContent(path)
	return content, err
}

// Which implements LanguageServerDelegate.Which
func (d *DefaultDelegate) Which(command string) (string, error) {
	path, err := exec.LookPath(command)
	return path, err
}

// ShellEnv implements LanguageServerDelegate.ShellEnv
func (d *DefaultDelegate) ShellEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i >= 0 {
			env[e[:i]] = e[i+1:]
		}
	}
	return env
}

// WorkspaceRoot implements LanguageServerDelegate.WorkspaceRoot
func (d *DefaultDelegate) WorkspaceRoot() string {
	return d.workspaceRoot
}