package defaults

import "time"

const (
	// Config defaults
	OutputFolder = "dataset"
	HTTPTimeout  = 300

	// System constants
	SystemStepName = "SYSTEM"

	// Provider URLs
	OllamaURL     = "http://localhost:11434/v1"
	LMStudioURL   = "http://127.0.0.1:1234/v1"
	OpenAIURL     = "https://api.openai.com/v1"
	OpenRouterURL = "https://openrouter.ai/api/v1"
	GeminiURL     = "https://generativelanguage.googleapis.com/v1beta/openai/"

	// Timeouts
	CmdTimeout        = 1 * time.Hour
	RetryInitialDelay = 1 * time.Second
	RetryMaxDelay     = 10 * time.Second

	// File system
	FileExtension = ".jsonl"
	FolderPerm    = 0o755

	// Version
	SupportedConfigVersion = "1.0"
)
