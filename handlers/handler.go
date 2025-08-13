package handlers

import (
	"context"
	"encoding/json"
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

// AIResponse matches the JSON structure we expect from the AI.
type AIResponse struct {
	Story           string   `json:"story"`
	Items           []string `json:"items"`
	ItemsRemoved    []string `json:"items_removed"`
	GameOver        bool     `json:"game_over"`
	BackgroundColor string   `json:"background_color"`
}

var (
	storyHistory []story.StoryPage
	inventory    []string
)

// parseAIResponse unmarshals the JSON from the AI.
func parseAIResponse(response string) (AIResponse, error) {
	var aiResp AIResponse
	// The AI might sometimes wrap the JSON in markdown, so we clean it first.
	cleanResponse := strings.TrimPrefix(response, "```json\n")
	cleanResponse = strings.TrimSuffix(cleanResponse, "\n```")

	err := json.Unmarshal([]byte(cleanResponse), &aiResp)
	return aiResp, err
}

func (h *Handler) StartStory(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Model.GenerateContent(context.Background(), genai.Text(prompts.SystemPrompt))
	if err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		http.Error(w, "The AI failed to start the story. Please try again.", http.StatusInternalServerError)
		return
	}

	aiResp, err := parseAIResponse(string(resp.Candidates[0].Content.Parts[0].(genai.Text)))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse AI's initial response: %v", err), http.StatusInternalServerError)
		return
	}

	// Reset inventory and story history for a new game.
	inventory = aiResp.Items
	storyHistory = []story.StoryPage{{Prompt: "Start", Response: aiResp.Story}}

	templates.StoryView(aiResp.Story, inventory, aiResp.BackgroundColor).Render(context.Background(), w)
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	prompt := r.FormValue("prompt")

	// Handle the restart command.
	if strings.ToLower(strings.TrimSpace(prompt)) == "restart" {
		h.StartStory(w, r)
		return
	}

	if len(strings.Fields(prompt)) > 15 {
		http.Error(w, "Response must be 15 words or less.", http.StatusBadRequest)
		return
	}

	fullStory := prompts.SystemPrompt
	for _, page := range storyHistory {
		fullStory += fmt.Sprintf("%s\n%s\n", page.Prompt, page.Response)
	}
	fullStory += fmt.Sprintf("%s\n", prompt)

	resp, err := h.Model.GenerateContent(r.Context(), genai.Text(fullStory))
	if err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		errorPage := story.StoryPage{Prompt: prompt, Response: "[The AI's response was blocked. Try something else.]"}
		storyHistory = append(storyHistory, errorPage)
		templates.Update(storyHistory, inventory, "#1e1e1e").Render(context.Background(), w) // Default color on error
		return
	}

	aiResp, err := parseAIResponse(string(resp.Candidates[0].Content.Parts[0].(genai.Text)))
	if err != nil {
		errorPage := story.StoryPage{Prompt: prompt, Response: fmt.Sprintf("[The AI's response was not valid JSON: %v]", err)}
		storyHistory = append(storyHistory, errorPage)
		templates.Update(storyHistory, inventory, "#1e1e1e").Render(context.Background(), w) // Default color on error
		return
	}

	// If the game is over, don't process inventory changes.
	if !aiResp.GameOver {
		itemsToRemove := make(map[string]bool)
		for _, item := range aiResp.ItemsRemoved {
			itemsToRemove[item] = true
		}

		var newInventory []string
		for _, item := range inventory {
			if !itemsToRemove[item] {
				newInventory = append(newInventory, item)
			}
		}
		inventory = newInventory
		inventory = append(inventory, aiResp.Items...)
	}

	storyHistory = append(storyHistory, story.StoryPage{Prompt: prompt, Response: aiResp.Story})
	templates.Update(storyHistory, inventory, aiResp.BackgroundColor).Render(context.Background(), w)
}