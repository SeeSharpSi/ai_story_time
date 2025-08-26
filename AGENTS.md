# AGENTS.md - Development Guidelines for Story AI

## Build/Lint/Test Commands

### Build Commands
- **Build application**: `go build -o story_ai .`
- **Run application**: `go run .`
- **Generate templates**: `templ generate`
- **Docker build**: `docker build -t story_ai .`

### Test Commands
- **Run all tests**: `go test ./...`
- **Run specific package tests**: `go test ./handlers`
- **Run single test**: `go test -run TestAddTooltipSpans ./handlers`
- **Run tests with verbose output**: `go test -v ./...`
- **Run tests with coverage**: `go test -cover ./...`

### Lint Commands
- **Format code**: `go fmt ./...`
- **Vet code**: `go vet ./...`
- **Run golangci-lint** (if installed): `golangci-lint run`

## Code Style Guidelines

### General Go Conventions
- Follow standard Go formatting with `go fmt`
- Use `go vet` for static analysis
- Write clear, descriptive variable and function names
- Use single responsibility principle for functions
- Keep functions under 50 lines when possible

### Imports
```go
import (
    // Standard library imports
    "context"
    "encoding/json"
    "log"
    "net/http"
    "strings"

    // Third-party imports
    "github.com/google/generative-ai-go/genai"
    "github.com/joho/godotenv"

    // Local imports
    "story_ai/handlers"
    "story_ai/session"
    "story_ai/story"
)
```

### Naming Conventions
- **Packages**: lowercase, single word (e.g., `handlers`, `story`, `session`)
- **Functions/Methods**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase, descriptive names
- **Constants**: PascalCase
- **Types**: PascalCase for exported types

### Struct Tags and JSON
```go
type GameState struct {
    PlayerStatus PlayerStatus `json:"player_status"`
    Inventory    []Item       `json:"inventory"`
    Environment  Environment  `json:"environment"`
    // Use snake_case for JSON tags to match API expectations
}
```

### Error Handling
- Always handle errors appropriately
- Use descriptive error messages
- Return errors from functions rather than panicking
- Log errors with context when appropriate

### HTTP Handlers
- Use proper HTTP status codes
- Set appropriate Content-Type headers
- Handle both success and error cases
- Validate input parameters

### Database Operations
- Always close database connections with `defer db.Close()`
- Use prepared statements for security
- Handle SQL errors appropriately
- Use transactions for multiple related operations

### Testing
- Write table-driven tests following the pattern in `handler_test.go`
- Use descriptive test names with `TestFunctionName`
- Test both success and error cases
- Use `t.Errorf` for clear error messages

### Comments
- Add comments for exported functions and types
- Use `//` for single-line comments
- Keep comments concise and descriptive
- No need for comments on obvious code

### File Organization
- Keep related functionality in the same package
- Use clear package names that reflect functionality
- Separate concerns (handlers, templates, static files)
- Follow Go project layout conventions

### Security
- Never log sensitive information
- Validate all user inputs
- Use environment variables for secrets
- Follow Go security best practices

### Performance
- Be mindful of memory usage
- Use efficient data structures
- Avoid unnecessary allocations
- Profile code when optimizing

### Known Issues to Address
- **Deprecated strings.Title**: Replace with `golang.org/x/text/cases` for proper Unicode handling (used in lines 602 and 609 of handler.go)
- **Broken test in handler_test.go**: The `TestAddTooltipSpans` function references `addTooltipSpans(tc.storyText, tc.nouns)` but this function doesn't exist. Either implement the function or remove/comment out the test.

## Project-Specific Patterns

### AI Integration
- Use structured prompts with clear instructions
- Handle AI response parsing robustly
- Implement retry logic for failed AI calls
- Validate AI responses before processing

### Session Management
- Use secure session handling
- Clean up old sessions
- Store session data efficiently
- Handle concurrent access safely

### Template Usage
- Use templ for type-safe HTML generation
- Keep templates clean and maintainable
- Use proper escaping for user content
- Follow consistent naming for template functions

### PDF Generation
- Handle HTML parsing carefully
- Preserve formatting in PDF output
- Add proper error handling for PDF generation
- Optimize PDF size when possible