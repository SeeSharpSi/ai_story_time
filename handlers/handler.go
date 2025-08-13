package handlers

import (
	"context"
	"fmt"
	"net/http"
	"story_ai/prompts"
	"story_ai/story"
	"story_ai/templates"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

type Handler struct {
	Model *genai.GenerativeModel
}

var storyHistory []story.StoryPage

func (h *Handler) StartStory(w http.ResponseWriter, r *http.Request) {
	// The initial prompt to the AI is just the system prompt.
	resp, err := h.Model.GenerateContent(context.Background(), genai.Text(prompts.SystemPrompt))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating initial content: %v", err), http.StatusInternalServerError)
		return
	}
	initialContent := string(resp.Candidates[0].Content.Parts[0].(genai.Text))
	// Clear history and add the first page.
	storyHistory = []story.StoryPage{{Prompt: "Start", Response: initialContent}}

	// Render the main story view
	templates.StoryView(initialContent).Render(context.Background(), w)
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	prompt := r.FormValue("prompt")
	words := strings.Fields(prompt)
	if len(words) > 15 {
		http.Error(w, "Response must be 15 words or less.", http.StatusBadRequest)
		return
	}

	// Show loading indicator
	fmt.Fprint(w, `<div id="loading" class="htmx-indicator">Loading...</div>`)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Construct the full story context for the AI
	fullStory := prompts.SystemPrompt
	for _, page := range storyHistory {
		fullStory += fmt.Sprintf("%s\n%s\n", page.Prompt, page.Response)
	}
	fullStory += fmt.Sprintf("%s\n", prompt)

	resp, err := h.Model.GenerateContent(r.Context(), genai.Text(fullStory))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating content: %v", err), http.StatusInternalServerError)
		return
	}

	newResponse := string(resp.Candidates[0].Content.Parts[0].(genai.Text))
	storyHistory = append(storyHistory, story.StoryPage{Prompt: prompt, Response: newResponse})

	// Render the updated story history
	templates.StoryPage(storyHistory).Render(context.Background(), w)
}