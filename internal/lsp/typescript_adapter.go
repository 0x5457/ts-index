package lsp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	tsLanguageServerNotInstalledMsg = `typescript-language-server is not installed. 
		Use 'ts-index lsp install typescript-language-server'`
)

// IsVTSLSInstalled checks if vtsls is installed and available
func IsVTSLSInstalled() bool {
	_, err := exec.LookPath("vtsls")
	return err == nil
}

// InstallVTSLSCommand returns the command to install vtsls
func InstallVTSLSCommand() string {
	return "npm install -g @vtsls/language-server"
}

// IsTypeScriptLanguageServerInstalled checks if typescript-language-server is installed and available
func IsTypeScriptLanguageServerInstalled() bool {
	_, err := exec.LookPath("typescript-language-server")
	return err == nil
}

// InstallTypeScriptLanguageServerCommand returns the command to install typescript-language-server
func InstallTypeScriptLanguageServerCommand() string {
	return "npm install -g typescript-language-server typescript"
}

// TypeScriptLspAdapter implements LspAdapter for TypeScript/JavaScript
type TypeScriptLspAdapter struct {
	serverType          ServerType
	installationManager *InstallationManager
}

type ServerType int

const (
	ServerTypeVTSLS ServerType = iota
	ServerTypeTypeScriptLanguageServer
)

// NewTypeScriptLspAdapter creates a new TypeScript LSP adapter
func NewTypeScriptLspAdapter() *TypeScriptLspAdapter {
	adapter := &TypeScriptLspAdapter{
		installationManager: NewInstallationManager(""),
	}

	// Determine which server to use
	if IsVTSLSInstalled() {
		adapter.serverType = ServerTypeVTSLS
	} else if IsTypeScriptLanguageServerInstalled() {
		adapter.serverType = ServerTypeTypeScriptLanguageServer
	} else {
		// Default to vtsls, installation will be handled separately
		adapter.serverType = ServerTypeVTSLS
	}

	return adapter
}

// NewTypeScriptLspAdapterWithInstallDir creates a new TypeScript LSP adapter with custom install directory
func NewTypeScriptLspAdapterWithInstallDir(installDir string) *TypeScriptLspAdapter {
	adapter := &TypeScriptLspAdapter{
		installationManager: NewInstallationManager(installDir),
		serverType:          ServerTypeVTSLS, // Default to vtsls
	}

	return adapter
}

// Name implements LspAdapter.Name
func (a *TypeScriptLspAdapter) Name() string {
	switch a.serverType {
	case ServerTypeVTSLS:
		return "vtsls"
	case ServerTypeTypeScriptLanguageServer:
		return "typescript-language-server"
	default:
		return typescriptLangName
	}
}

// LanguageIds implements LspAdapter.LanguageIds
func (a *TypeScriptLspAdapter) LanguageIds() map[string]string {
	return map[string]string{
		typescriptLangName: typescriptLangName,
		"javascript":       "javascript",
		"typescriptreact":  "typescriptreact",
		"javascriptreact":  "javascriptreact",
		"tsx":              "typescriptreact",
		"jsx":              "javascriptreact",
		"ts":               typescriptLangName,
		"js":               "javascript",
	}
}

// ServerCommand implements LspAdapter.ServerCommand
func (a *TypeScriptLspAdapter) ServerCommand(workspaceRoot string) (string, []string, error) {
	delegate := NewDefaultDelegate(workspaceRoot)

	// First try to get from local installation
	serverName := a.Name()
	if binary, err := a.installationManager.GetServerBinary(serverName, "", delegate); err == nil &&
		binary != nil {
		return binary.Path, binary.Args, nil
	}

	// Fallback to system-wide installation
	switch a.serverType {
	case ServerTypeVTSLS:
		if IsVTSLSInstalled() {
			return "vtsls", []string{"--stdio"}, nil
		}
		return "", nil, fmt.Errorf(
			"vtsls is not installed. Use 'ts-index lsp install vtsls' or install globally with: %s",
			InstallVTSLSCommand(),
		)

	case ServerTypeTypeScriptLanguageServer:
		if IsTypeScriptLanguageServerInstalled() {
			return "typescript-language-server", []string{"--stdio"}, nil
		}
		return "", nil, fmt.Errorf(
			"%s or install globally with: %s",
			tsLanguageServerNotInstalledMsg,
			InstallTypeScriptLanguageServerCommand(),
		)

	default:
		return "", nil, fmt.Errorf("unknown server type")
	}
}

// InitializationOptions implements LspAdapter.InitializationOptions
func (a *TypeScriptLspAdapter) InitializationOptions(
	workspaceRoot string,
) (map[string]interface{}, error) {
	switch a.serverType {
	case ServerTypeVTSLS:
		return map[string]interface{}{
			"typescript": map[string]interface{}{
				"suggest": map[string]interface{}{
					"autoImports": true,
				},
				"inlayHints": map[string]interface{}{
					"includeInlayParameterNameHints":                        "all",
					"includeInlayParameterNameHintsWhenArgumentMatchesName": false,
					"includeInlayFunctionParameterTypeHints":                true,
					"includeInlayVariableTypeHints":                         false,
					"includeInlayPropertyDeclarationTypeHints":              true,
					"includeInlayFunctionLikeReturnTypeHints":               true,
					"includeInlayEnumMemberValueHints":                      true,
				},
			},
			"vtsls": map[string]interface{}{
				"experimental": map[string]interface{}{
					"completion": map[string]interface{}{
						"enableServerSideFuzzyMatch": true,
					},
				},
			},
		}, nil

	case ServerTypeTypeScriptLanguageServer:
		return map[string]interface{}{
			"preferences": map[string]interface{}{
				"includeCompletionsForModuleExports": true,
				"includeCompletionsWithInsertText":   true,
			},
		}, nil

	default:
		return nil, nil
	}
}

// WorkspaceConfiguration implements LspAdapter.WorkspaceConfiguration
func (a *TypeScriptLspAdapter) WorkspaceConfiguration(
	workspaceRoot string,
) (map[string]interface{}, error) {
	// Check for tsconfig.json
	tsconfigPath := filepath.Join(workspaceRoot, "tsconfig.json")
	if _, err := os.Stat(tsconfigPath); err == nil {
		// Project has TypeScript configuration
		return map[string]interface{}{
			"typescript": map[string]interface{}{
				"preferences": map[string]interface{}{
					"includePackageJsonAutoImports": "auto",
				},
			},
		}, nil
	}

	// Default JavaScript configuration
	return map[string]interface{}{
		"javascript": map[string]interface{}{
			"preferences": map[string]interface{}{
				"includePackageJsonAutoImports": "auto",
			},
		},
	}, nil
}

// ProcessDiagnostics implements LspAdapter.ProcessDiagnostics
func (a *TypeScriptLspAdapter) ProcessDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	// Filter out certain diagnostics or modify them
	var filtered []Diagnostic

	for _, diag := range diagnostics {
		// Skip unnecessary diagnostics for certain cases
		if shouldIncludeDiagnostic(diag) {
			filtered = append(filtered, diag)
		}
	}

	return filtered
}

// ProcessCompletions implements LspAdapter.ProcessCompletions
func (a *TypeScriptLspAdapter) ProcessCompletions(items []CompletionItem) []CompletionItem {
	// Enhance or filter completion items
	for i := range items {
		// Add additional information or modify existing items
		a.enhanceCompletionItem(&items[i])
	}

	return items
}

// CanInstall implements LspAdapter.CanInstall
func (a *TypeScriptLspAdapter) CanInstall() bool {
	// Check if npm is available for installation
	_, err := exec.LookPath("npm")
	return err == nil
}

// Install implements LspAdapter.Install
func (a *TypeScriptLspAdapter) Install(ctx context.Context) error {
	if !a.CanInstall() {
		return fmt.Errorf("npm is not available for installation")
	}

	delegate := &SimpleDelegate{}
	serverName := a.Name()

	// Install to local directory
	_, err := a.installationManager.InstallServer(ctx, serverName, "", delegate)
	return err
}

// InstallVersion installs a specific version of the language server
func (a *TypeScriptLspAdapter) InstallVersion(ctx context.Context, version string) error {
	if !a.CanInstall() {
		return fmt.Errorf("npm is not available for installation")
	}

	delegate := &SimpleDelegate{}
	serverName := a.Name()

	// Install specific version to local directory
	_, err := a.installationManager.InstallServer(ctx, serverName, version, delegate)
	return err
}

// IsInstalled implements LspAdapter.IsInstalled
func (a *TypeScriptLspAdapter) IsInstalled() bool {
	delegate := &SimpleDelegate{}
	serverName := a.Name()

	// Check local installation first
	if binary, err := a.installationManager.GetServerBinary(serverName, "", delegate); err == nil &&
		binary != nil {
		return true
	}

	// Check system-wide installation
	switch a.serverType {
	case ServerTypeVTSLS:
		return IsVTSLSInstalled()
	case ServerTypeTypeScriptLanguageServer:
		return IsTypeScriptLanguageServerInstalled()
	default:
		return false
	}
}

// Helper functions
// unparam
func shouldIncludeDiagnostic(diag Diagnostic) bool { //nolint:unparam
	// Skip certain noisy diagnostics
	if diag.Source != nil {
		source := *diag.Source
		if source == "ts" {
			// Skip certain TypeScript diagnostics that are too noisy
			// This is where you'd implement custom filtering logic
			// For now, we include all TypeScript diagnostics
			return true
		}
	}
	return true
}

func (a *TypeScriptLspAdapter) enhanceCompletionItem(item *CompletionItem) {
	// Add custom enhancements to completion items
	// For example, improve documentation or add custom sorting

	// Enhance detail information
	if item.Detail == nil || *item.Detail == "" {
		if item.Kind != nil {
			detail := getCompletionKindDescription(*item.Kind)
			item.Detail = &detail
		}
	}
}

func getCompletionKindDescription(kind CompletionKind) string {
	switch kind {
	case CompletionKindFunction:
		return "function"
	case CompletionKindMethod:
		return "method"
	case CompletionKindClass:
		return "class"
	case CompletionKindInterface:
		return "interface"
	case CompletionKindVariable:
		return "variable"
	case CompletionKindProperty:
		return "property"
	default:
		return ""
	}
}
