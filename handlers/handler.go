package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"story_ai/prompts"
	"story_ai/story"
	"story_ai/templates"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/jung-kurt/gofpdf"
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
	currentGenre string
	currentAuthor string
	authors = []string{"William Faulkner", "James Joyce", "Mark Twain", "Jack Kerouac", "Kurt Vonnegut", "Other"}
)

// parseAIResponse unmarshals the JSON from the AI.
func parseAIResponse(response string) (AIResponse, error) {
	var aiResp AIResponse
	cleanResponse := strings.TrimPrefix(response, "```json\n")
	cleanResponse = strings.TrimSuffix(cleanResponse, "\n```")

	err := json.Unmarshal([]byte(cleanResponse), &aiResp)
	return aiResp, err
}

func (h *Handler) StartStory(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query().Get("genre")
	
	rand.Seed(time.Now().UnixNano())
	author := authors[rand.Intn(len(authors))]

	if author == "Other" {
		authorPrompt := "Name one famous author who is not on this list: William Faulkner, James Joyce, Mark Twain, Jack Kerouac, Kurt Vonnegut. Respond with only the author's name."
		resp, err := h.Model.GenerateContent(context.Background(), genai.Text(authorPrompt))
		if err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			author = "Mark Twain"
		} else {
			author = string(resp.Candidates[0].Content.Parts[0].(genai.Text))
		}
	}
	currentAuthor = author

	var prompt string
	switch genre {
	case "fantasy":
		prompt = fmt.Sprintf(prompts.FantasyPrompt, currentAuthor)
		currentGenre = "fantasy"
	case "sci-fi":
		prompt = fmt.Sprintf(prompts.SciFiPrompt, currentAuthor)
		currentGenre = "sci-fi"
	case "historical-fiction":
		prompt = fmt.Sprintf(prompts.HistoricalFictionPrompt, currentAuthor)
		currentGenre = "historical-fiction"
	default:
		prompt = fmt.Sprintf(prompts.FantasyPrompt, currentAuthor)
		currentGenre = "fantasy"
	}

	resp, err := h.Model.GenerateContent(context.Background(), genai.Text(prompt))
	if err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		http.Error(w, "The AI failed to start the story. Please try again.", http.StatusInternalServerError)
		return
	}

	aiResp, err := parseAIResponse(string(resp.Candidates[0].Content.Parts[0].(genai.Text)))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse AI's initial response: %v", err), http.StatusInternalServerError)
		return
	}

	if aiResp.BackgroundColor == "" {
		aiResp.BackgroundColor = "#1e1e1e"
	}

	inventory = aiResp.Items
	storyHistory = []story.StoryPage{{Prompt: "Start", Response: aiResp.Story}}

	templates.StoryView(aiResp.Story, inventory, aiResp.BackgroundColor).Render(context.Background(), w)
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	prompt := r.FormValue("prompt")

	if strings.ToLower(strings.TrimSpace(prompt)) == "restart" {
		r.URL.RawQuery = "genre=" + currentGenre
		h.StartStory(w, r)
		return
	}

	if len(strings.Fields(prompt)) > 15 {
		http.Error(w, "Response must be 15 words or less.", http.StatusBadRequest)
		return
	}

	var systemPrompt string
	switch currentGenre {
	case "fantasy":
		systemPrompt = fmt.Sprintf(prompts.FantasyPrompt, currentAuthor)
	case "sci-fi":
		systemPrompt = fmt.Sprintf(prompts.SciFiPrompt, currentAuthor)
	case "historical-fiction":
		systemPrompt = fmt.Sprintf(prompts.HistoricalFictionPrompt, currentAuthor)
	default:
		systemPrompt = fmt.Sprintf(prompts.FantasyPrompt, currentAuthor)
	}

	var historyBuilder strings.Builder
	historyBuilder.WriteString(systemPrompt)
	for _, page := range storyHistory {
		historyBuilder.WriteString(fmt.Sprintf("%s\n%s\n", page.Prompt, page.Response))
	}
	historyBuilder.WriteString(fmt.Sprintf("%s\n", prompt))
	fullStory := historyBuilder.String()

	resp, err := h.Model.GenerateContent(r.Context(), genai.Text(fullStory))
	if err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		errorPage := story.StoryPage{Prompt: prompt, Response: "[The AI's response was blocked. Try something else.]"}
		storyHistory = append(storyHistory, errorPage)
		templates.Update(storyHistory, inventory, "#1e1e1e", false).Render(context.Background(), w)
		return
	}

	aiResp, err := parseAIResponse(string(resp.Candidates[0].Content.Parts[0].(genai.Text)))
	if err != nil {
		errorPage := story.StoryPage{Prompt: prompt, Response: fmt.Sprintf("[The AI's response was not valid JSON: %v]", err)}
		storyHistory = append(storyHistory, errorPage)
		templates.Update(storyHistory, inventory, "#1e1e1e", false).Render(context.Background(), w)
		return
	}
	
	if aiResp.BackgroundColor == "" {
		aiResp.BackgroundColor = "#1e1e1e"
	}

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
	templates.Update(storyHistory, inventory, aiResp.BackgroundColor, aiResp.GameOver).Render(context.Background(), w)
}

func (h *Handler) DownloadStory(w http.ResponseWriter, r *http.Request) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Your Story")
	pdf.Ln(20)

	pdf.SetFont("Arial", "", 12)
	for _, page := range storyHistory {
		// User prompt
		pdf.SetFontStyle("I")
		pdf.Write(5, "You: "+page.Prompt)
		pdf.Ln(10)

		// AI response
		pdf.SetFontStyle("")
		// We need to decode the HTML entities for the PDF
		cleanResponse := strings.ReplaceAll(page.Response, `<span class="item-added">`, "")
		cleanResponse = strings.ReplaceAll(cleanResponse, `<span class="item-removed">`, "")
		cleanResponse = strings.ReplaceAll(cleanResponse, `</span>`, "")
		cleanResponse = strings.ReplaceAll(cleanResponse, `<strong>`, "")
		cleanResponse = strings.ReplaceAll(cleanResponse, `</strong>`, "")
		pdf.MultiCell(0, 5, cleanResponse, "", "", false)
		pdf.Ln(10)
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=story.pdf")
	err := pdf.Output(w)
	if err != nil {
		http.Error(w, "Failed to generate PDF.", http.StatusInternalServerError)
	}
}