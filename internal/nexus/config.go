package nexus

import (
	"context"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

// ConfigError represents domain-specific configuration errors
type ConfigError struct {
	Code    string
	Message string
	Field   string
	Cause   error
}

func (e ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("[%s] %s (field: %s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e ConfigError) Unwrap() error {
	return e.Cause
}

// Common error codes for internationalization
const (
	ErrCodeInvalidType   = "CONFIG_INVALID_TYPE"
	ErrCodeFileNotFound  = "CONFIG_FILE_NOT_FOUND"
	ErrCodeValidation    = "CONFIG_VALIDATION_FAILED"
	ErrCodeEnvironment   = "CONFIG_ENV_READ_FAILED"
	ErrCodeMerge         = "CONFIG_MERGE_FAILED"
	ErrCodeSourceFailed  = "CONFIG_SOURCE_FAILED"
	ErrCodeSecurityCheck = "CONFIG_SECURITY_CHECK_FAILED"
)

// Source represents a configuration source
type Source interface {
	Load(ctx context.Context, target interface{}) error
	Name() string
	Priority() int
}

// Validator handles configuration validation
type Validator interface {
	Validate(ctx context.Context, cfg interface{}) error
}

// SecurityChecker performs security validation on configuration
type SecurityChecker interface {
	CheckSecurity(ctx context.Context, cfg interface{}) error
}

// I18nProvider handles internationalization
type I18nProvider interface {
	Translate(ctx context.Context, key string, args ...interface{}) string
}

// LoaderOptions contains configuration for the loader
type LoaderOptions struct {
	DefaultFileName string
	FileFlag        string
	FileName        string
	OnlyEnvironment bool
	Validator       Validator
	SecurityChecker SecurityChecker
	I18nProvider    I18nProvider
	Sources         []Source
	Timeout         time.Duration
}

// Loader represents a modular configuration loader
type Loader struct {
	options LoaderOptions
}

// LoaderOption is a functional option for configuring the loader
type LoaderOption func(*LoaderOptions)

// WithDefaultFileName sets the default configuration file name
func WithDefaultFileName(fileName string) LoaderOption {
	return func(o *LoaderOptions) {
		o.DefaultFileName = fileName
	}
}

// WithFileFlag sets the command line flag for configuration file
func WithFileFlag(flag string) LoaderOption {
	return func(o *LoaderOptions) {
		o.FileFlag = flag
		o.FileName = ""
	}
}

// WithFileName sets a specific configuration file name
func WithFileName(fileName string) LoaderOption {
	return func(o *LoaderOptions) {
		o.FileName = fileName
		o.FileFlag = ""
	}
}

// WithOnlyEnvironment configures loader to only read from environment
func WithOnlyEnvironment() LoaderOption {
	return func(o *LoaderOptions) {
		o.OnlyEnvironment = true
		o.FileFlag = ""
		o.FileName = ""
	}
}

// WithValidator sets a custom validator
func WithValidator(v Validator) LoaderOption {
	return func(o *LoaderOptions) {
		o.Validator = v
	}
}

// WithSecurityChecker sets a custom security checker
func WithSecurityChecker(sc SecurityChecker) LoaderOption {
	return func(o *LoaderOptions) {
		o.SecurityChecker = sc
	}
}

// WithI18nProvider sets an internationalization provider
func WithI18nProvider(provider I18nProvider) LoaderOption {
	return func(o *LoaderOptions) {
		o.I18nProvider = provider
	}
}

// WithSources adds custom configuration sources
func WithSources(sources ...Source) LoaderOption {
	return func(o *LoaderOptions) {
		o.Sources = append(o.Sources, sources...)
	}
}

// WithTimeout sets the timeout for loading operations
func WithTimeout(timeout time.Duration) LoaderOption {
	return func(o *LoaderOptions) {
		o.Timeout = timeout
	}
}

// NewLoader creates a new configuration loader with options
func NewLoader(opts ...LoaderOption) *Loader {
	options := LoaderOptions{
		DefaultFileName: ".env",
		FileFlag:        "config",
		Validator:       &DefaultValidator{},
		SecurityChecker: &DefaultSecurityChecker{},
		I18nProvider:    &DefaultI18nProvider{},
		Timeout:         30 * time.Second,
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &Loader{options: options}
}

// Load loads configuration from all configured sources
func (l *Loader) Load(cfg interface{}) error {
	return l.LoadWithContext(context.Background(), cfg)
}

// LoadWithContext loads configuration with context support
func (l *Loader) LoadWithContext(ctx context.Context, cfg interface{}) error {
	// Create context with timeout
	if l.options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, l.options.Timeout)
		defer cancel()
	}

	// Validate input type
	if err := l.validateInputType(cfg); err != nil {
		return err
	}

	// Load from built-in sources
	if err := l.loadFromBuiltinSources(ctx, cfg); err != nil {
		return err
	}

	// Load from custom sources
	if err := l.loadFromCustomSources(ctx, cfg); err != nil {
		return err
	}

	// Perform security checks
	if err := l.options.SecurityChecker.CheckSecurity(ctx, cfg); err != nil {
		return &ConfigError{
			Code:    ErrCodeSecurityCheck,
			Message: l.options.I18nProvider.Translate(ctx, "security_check_failed"),
			Cause:   err,
		}
	}

	// Validate final configuration
	if err := l.options.Validator.Validate(ctx, cfg); err != nil {
		return &ConfigError{
			Code:    ErrCodeValidation,
			Message: l.options.I18nProvider.Translate(ctx, "validation_failed"),
			Cause:   err,
		}
	}

	return nil
}

func (l *Loader) validateInputType(cfg interface{}) error {
	if reflect.ValueOf(cfg).Kind() != reflect.Ptr {
		return &ConfigError{
			Code:    ErrCodeInvalidType,
			Message: fmt.Sprintf("configuration must be a pointer to struct, got %T", cfg),
		}
	}
	return nil
}

func (l *Loader) loadFromBuiltinSources(ctx context.Context, cfg interface{}) error {
	// Load from environment variables
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return &ConfigError{
			Code:    ErrCodeEnvironment,
			Message: l.options.I18nProvider.Translate(ctx, "env_read_failed"),
			Cause:   err,
		}
	}

	// Load from file if not environment-only
	if !l.options.OnlyEnvironment {
		fileName := l.resolveFileName()
		if fileName != "" {
			if err := l.loadFromFile(ctx, cfg, fileName); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *Loader) loadFromFile(ctx context.Context, cfg interface{}, fileName string) error {
	// Create a copy for file configuration
	fileCfg := reflect.New(reflect.ValueOf(cfg).Elem().Type()).Interface()

	if err := cleanenv.ReadConfig(fileName, fileCfg); err != nil {
		return &ConfigError{
			Code:    ErrCodeFileNotFound,
			Message: l.options.I18nProvider.Translate(ctx, "file_read_failed", fileName),
			Cause:   err,
		}
	}

	// Merge with environment variables taking precedence
	if err := mergo.MergeWithOverwrite(cfg, fileCfg); err != nil {
		return &ConfigError{
			Code:    ErrCodeMerge,
			Message: l.options.I18nProvider.Translate(ctx, "merge_failed"),
			Cause:   err,
		}
	}

	return nil
}

func (l *Loader) loadFromCustomSources(ctx context.Context, cfg interface{}) error {
	// Sort sources by priority
	sources := make([]Source, len(l.options.Sources))
	copy(sources, l.options.Sources)

	// Simple priority sort (can be enhanced with more sophisticated sorting)
	for i := 0; i < len(sources)-1; i++ {
		for j := i + 1; j < len(sources); j++ {
			if sources[i].Priority() < sources[j].Priority() {
				sources[i], sources[j] = sources[j], sources[i]
			}
		}
	}

	// Load from each source
	for _, source := range sources {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := source.Load(ctx, cfg); err != nil {
				return &ConfigError{
					Code:    ErrCodeSourceFailed,
					Message: l.options.I18nProvider.Translate(ctx, "source_failed", source.Name()),
					Cause:   err,
				}
			}
		}
	}

	return nil
}

func (l *Loader) resolveFileName() string {
	// Return explicit filename if set
	if l.options.FileName != "" {
		return l.options.FileName
	}

	// Skip flag handling if no flag configured
	if l.options.FileFlag == "" {
		return ""
	}

	// Get filename from flag
	fileName := l.getFileNameFromFlag()

	// Use default file if flag not specified but default exists
	if fileName == "" {
		fileName = l.getDefaultFileIfExists()
	}

	return fileName
}

// getFileNameFromFlag retrieves filename from command line flag
func (l *Loader) getFileNameFromFlag() string {
	f := flag.Lookup(l.options.FileFlag)
	if f != nil {
		return f.Value.String()
	}

	// Flag doesn't exist, create it
	var fileName string
	flag.StringVar(&fileName, l.options.FileFlag, "", "Specify configuration file")
	flag.Parse()
	return fileName
}

// getDefaultFileIfExists returns default filename if it exists
func (l *Loader) getDefaultFileIfExists() string {
	if l.options.DefaultFileName == "" {
		return ""
	}

	if _, err := os.Stat(l.options.DefaultFileName); err == nil {
		return l.options.DefaultFileName
	}

	return ""
}

// DefaultValidator implements basic validation using go-playground/validator
type DefaultValidator struct {
	validator *validator.Validate
}

func (v *DefaultValidator) Validate(_ context.Context, cfg interface{}) error {
	if v.validator == nil {
		v.validator = validator.New()
	}
	return v.validator.Struct(cfg)
}

// DefaultSecurityChecker implements basic security checks
type DefaultSecurityChecker struct{}

func (sc *DefaultSecurityChecker) CheckSecurity(_ context.Context, cfg interface{}) error {
	// Check for common security issues
	val := reflect.ValueOf(cfg).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Check for sensitive fields that might be exposed
		if sc.isSensitiveField(fieldType.Name) && field.Kind() == reflect.String {
			if sc.isValueExposed(field.String()) {
				return fmt.Errorf("sensitive field %s appears to contain exposed credentials", fieldType.Name)
			}
		}
	}

	return nil
}

func (sc *DefaultSecurityChecker) isSensitiveField(fieldName string) bool {
	sensitiveFields := []string{"password", "secret", "key", "token", "credential"}
	fieldLower := strings.ToLower(fieldName)

	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}

func (sc *DefaultSecurityChecker) isValueExposed(value string) bool {
	// Check for common patterns of exposed credentials
	exposedPatterns := []string{"password", "123456", "admin", "test"}
	valueLower := strings.ToLower(value)

	for _, pattern := range exposedPatterns {
		if strings.Contains(valueLower, pattern) {
			return true
		}
	}
	return false
}

// DefaultI18nProvider provides basic internationalization
type DefaultI18nProvider struct{}

func (i18n *DefaultI18nProvider) Translate(_ context.Context, key string, args ...interface{}) string {
	// Basic English messages - in a real implementation, this would load from translation files
	messages := map[string]string{
		"security_check_failed": "Security validation failed",
		"validation_failed":     "Configuration validation failed",
		"env_read_failed":       "Failed to read environment variables",
		"file_read_failed":      "Failed to read configuration file: %s",
		"merge_failed":          "Failed to merge configuration sources",
		"source_failed":         "Failed to load from source: %s",
	}

	if msg, ok := messages[key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}
	return key
}

// FileSource implements Source interface for file-based configuration
type FileSource struct {
	FilePath string
	priority int
}

func NewFileSource(filePath string, priority int) *FileSource {
	return &FileSource{
		FilePath: filePath,
		priority: priority,
	}
}

func (fs *FileSource) Load(_ context.Context, target interface{}) error {
	return cleanenv.ReadConfig(fs.FilePath, target)
}

func (fs *FileSource) Name() string {
	return fmt.Sprintf("file:%s", fs.FilePath)
}

func (fs *FileSource) Priority() int {
	return fs.priority
}
