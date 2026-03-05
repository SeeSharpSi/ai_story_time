# AGENTS.md - Development Guidelines for Story AI

## Build/Lint/Test Commands

### Build Commands
- **Build application**: `go build -o story_ai .`
- **Run application**: `go run .`
- **Generate templates**: `templ generate` (required before running)
- **Docker build**: `docker build -t story_ai .`
- **Deploy to Cloud Run**: `./deploy.sh` (requires GEMINI_API_KEY)

### Test Commands
- **Run all tests**: `go test ./...`
- **Run specific package tests**: `go test ./handlers`
- **Run single test**: `go test -run TestFunctionName ./packagename`
- **Run tests with verbose output**: `go test -v ./...`
- **Run tests with coverage**: `go test -cover ./...`
- **Run tests with coverage report**: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`

### Lint Commands
- **Format code**: `go fmt ./...`
- **Vet code**: `go vet ./...`
- **Run golangci-lint** (if installed): `golangci-lint run`
- **Check for deprecated functions**: `go list -f '{{.Dir}}: {{.Imports}}' ./...`

## Code Style Guidelines

### General Go Conventions
- Follow standard Go formatting with `go fmt`, use `go vet` for static analysis
- Write clear, descriptive names using single responsibility principle
- **Packages**: lowercase, single word; **Functions**: PascalCase for exported, camelCase for unexported
- **Variables**: camelCase; **Constants**: PascalCase; **Types**: PascalCase for exported
- Keep functions under 50 lines when possible

### Import Organization
```go
import (
    // Standard library
    "context" "encoding/json" "log" "net/http" "strings"
    
    // Third-party
    "github.com/google/generative-ai-go/genai"
    "github.com/joho/godotenv"
    
    // Local imports
    "story_ai/handlers" "story_ai/session" "story_ai/story"
)
```

### Error Handling & Security
- Always handle errors with descriptive messages, return rather than panic
- Never log sensitive info, validate all inputs, use environment variables for secrets
- For HTTP: proper status codes, Content-Type headers, input validation
- For database: `defer db.Close()`, prepared statements, transactions

### Testing & Documentation
- Write table-driven tests with `TestFunctionName` pattern, test success/error cases
- Add comments for exported functions/types, keep concise
- Use `t.Errorf` for clear test error messages

### Performance & File Organization
- Use efficient data structures, avoid unnecessary allocations, profile when optimizing
- Keep related functionality together, separate concerns, follow Go project layout

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

## Project Structure

```
story_ai/
├── main.go                 # Application entry point
├── go.mod/go.sum          # Go module dependencies
├── handlers/              # HTTP request handlers
│   ├── handler.go         # Main story generation logic
│   ├── handler_test.go    # Tests (currently empty)
│   ├── health.go          # Health check endpoints
│   ├── validation.go      # Input validation
│   ├── errors.go          # Error handling
│   └── fallback.go        # Fallback responses
├── story/                 # Story domain logic
│   ├── story.go           # Story data structures
│   └── state.go           # Game state management
├── session/               # Session management
│   └── session.go         # Session handling logic
├── templates/             # HTML templates (templ)
│   ├── index.templ        # Main page template
│   ├── story_view.templ   # Story display template
│   └── update.templ       # Dynamic update template
├── static/                # Static assets
│   ├── htmx.min.js        # HTMX library
│   └── favicon.jpg        # Site favicon
├── middleware/            # HTTP middleware
│   ├── csrf.go            # CSRF protection
│   ├── ratelimit.go       # Rate limiting
│   └── sizelimit.go       # Request size limits
├── prompts/               # AI prompt templates
│   └── prompts.go         # Story generation prompts
├── logger/                # Logging utilities
│   └── logger.go          # Application logger
├── metrics/               # Metrics collection
│   └── metrics.go         # Performance metrics
├── data.db               # SQLite database
├── Dockerfile            # Container configuration
├── deploy.sh             # Cloud deployment script
└── .env                  # Environment variables (local)
```

## Environment Setup

### Required Environment Variables
- `GEMINI_API_KEY`: Google Gemini API key (required)
- `PORT`: Application port (defaults to 9779)
- `METRICS_DATABASE_URL`: Metrics collection endpoint (defaults to http://localhost:8081)

### Development Workflow
1. Run `templ generate` after any template changes
2. Use `go run .` for local development
3. Test with `go test ./...`
4. Format code with `go fmt ./...`
5. Deploy with `./deploy.sh` (requires GCP setup)