package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"story_ai/prompts"
	"story_ai/session"
	"story_ai/story"
	"story_ai/templates"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/jung-kurt/gofpdf"
)

type Handler struct {
	Model   *genai.GenerativeModel
	Manager *session.Manager
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
	authors = []string{"William Faulkner", "James Joyce", "Mark Twain", "Jack Kerouac", "Kurt Vonnegut", "Other"}
	// Regex to find Markdown bolding (**text**)
	markdownBoldRegex = regexp.MustCompile(`\*\*(.*?)\*\*`)
	// Regex to find Markdown italics (*text*)
	markdownItalicRegex = regexp.MustCompile(`\*(.*?)\*`)
)

// parseAIResponse unmarshals the JSON from the AI and cleans up the story text.
func parseAIResponse(response string) (AIResponse, error) {
	var aiResp AIResponse
	cleanResponse := strings.TrimPrefix(response, "```json\n")
	cleanResponse = strings.TrimSuffix(cleanResponse, "\n```")

	err := json.Unmarshal([]byte(cleanResponse), &aiResp)
	if err != nil {
		return aiResp, err
	}

	// Failsafe: Replace any Markdown bolding with <strong> tags.
	aiResp.Story = markdownBoldRegex.ReplaceAllString(aiResp.Story, "<strong>$1</strong>")
	// Failsafe: Replace any Markdown italics with <em> tags.
	aiResp.Story = markdownItalicRegex.ReplaceAllString(aiResp.Story, "<em>$1</em>")

	return aiResp, nil
}

func (h *Handler) StartStory(w http.ResponseWriter, r *http.Request) {
	sess, cookie := h.Manager.GetOrCreateSession(r)
	http.SetCookie(w, &cookie)

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
	sess.CurrentAuthor = author

	var prompt string
	switch genre {
	case "fantasy":
		prompt = fmt.Sprintf(prompts.FantasyPrompt, sess.CurrentAuthor)
		sess.CurrentGenre = "fantasy"
	case "sci-fi":
		prompt = fmt.Sprintf(prompts.SciFiPrompt, sess.CurrentAuthor)
		sess.CurrentGenre = "sci-fi"
	case "historical-fiction":
		prompt = fmt.Sprintf(prompts.HistoricalFictionPrompt, sess.CurrentAuthor)
		sess.CurrentGenre = "historical-fiction"
	default:
		prompt = fmt.Sprintf(prompts.FantasyPrompt, sess.CurrentAuthor)
		sess.CurrentGenre = "fantasy"
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

	sess.Inventory = aiResp.Items
	sess.StoryHistory = []story.StoryPage{{Prompt: "Start", Response: aiResp.Story}}

	templates.StoryView(aiResp.Story, sess.Inventory, aiResp.BackgroundColor).Render(context.Background(), w)
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Manager.GetOrCreateSession(r)
	prompt := r.FormValue("prompt")

	if strings.ToLower(strings.TrimSpace(prompt)) == "restart" {
		r.URL.RawQuery = "genre=" + sess.CurrentGenre
		h.StartStory(w, r)
		return
	}

	if len(strings.Fields(prompt)) > 15 {
		http.Error(w, "Response must be 15 words or less.", http.StatusBadRequest)
		return
	}

	var systemPrompt string
	switch sess.CurrentGenre {
	case "fantasy":
		systemPrompt = fmt.Sprintf(prompts.FantasyPrompt, sess.CurrentAuthor)
	case "sci-fi":
		systemPrompt = fmt.Sprintf(prompts.SciFiPrompt, sess.CurrentAuthor)
	case "historical-fiction":
		systemPrompt = fmt.Sprintf(prompts.HistoricalFictionPrompt, sess.CurrentAuthor)
	default:
		systemPrompt = fmt.Sprintf(prompts.FantasyPrompt, sess.CurrentAuthor)
	}

	var historyBuilder strings.Builder
	historyBuilder.WriteString(systemPrompt)
	for _, page := range sess.StoryHistory {
		historyBuilder.WriteString(fmt.Sprintf("%s\n%s\n", page.Prompt, page.Response))
	}
	historyBuilder.WriteString(fmt.Sprintf("%s\n", prompt))
	fullStory := historyBuilder.String()

	resp, err := h.Model.GenerateContent(r.Context(), genai.Text(fullStory))
	if err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		errorPage := story.StoryPage{Prompt: prompt, Response: "[The AI's response was blocked. Try something else.]"}
		sess.StoryHistory = append(sess.StoryHistory, errorPage)
		templates.Update(sess.StoryHistory, sess.Inventory, "#1e1e1e", false, sess.CurrentGenre).Render(context.Background(), w)
		return
	}

	aiResp, err := parseAIResponse(string(resp.Candidates[0].Content.Parts[0].(genai.Text)))
	if err != nil {
		errorPage := story.StoryPage{Prompt: prompt, Response: fmt.Sprintf("[The AI's response was not valid JSON: %v]", err)}
		sess.StoryHistory = append(sess.StoryHistory, errorPage)
		templates.Update(sess.StoryHistory, sess.Inventory, "#1e1e1e", false, sess.CurrentGenre).Render(context.Background(), w)
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
		for _, item := range sess.Inventory {
			if !itemsToRemove[item] {
				newInventory = append(newInventory, item)
			}
		}
		sess.Inventory = newInventory
		sess.Inventory = append(sess.Inventory, aiResp.Items...)
	}

	sess.StoryHistory = append(sess.StoryHistory, story.StoryPage{Prompt: prompt, Response: aiResp.Story})
	templates.Update(sess.StoryHistory, sess.Inventory, aiResp.BackgroundColor, aiResp.GameOver, sess.CurrentGenre).Render(context.Background(), w)
}

func (h *Handler) DownloadStory(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Manager.GetOrCreateSession(r)
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 24)
	pdf.Cell(0, 10, "Your Story")
	pdf.Ln(15)

	pdf.SetFont("Helvetica", "I", 14)
	pdf.Cell(0, 10, "An AI-generated story in the style of "+sess.CurrentAuthor)
	pdf.Ln(20)

	pdf.SetFont("Helvetica", "", 12)
	for _, page := range sess.StoryHistory {
		pdf.SetFontStyle("I")
		pdf.Write(5, page.Prompt)
		pdf.Ln(10)

		pdf.SetFontStyle("")
		// Basic HTML tag removal
		cleanResponse := strings.ReplaceAll(page.Response, "<strong>", "")
		cleanResponse = strings.ReplaceAll(cleanResponse, "</strong>", "")
		cleanResponse = strings.ReplaceAll(cleanResponse, "<em>", "")
		cleanResponse = strings.ReplaceAll(cleanResponse, "</em>", "")
		cleanResponse = strings.ReplaceAll(cleanResponse, `<span class="item-added">`, "")
		cleanResponse = strings.ReplaceAll(cleanResponse, `<span class="item-removed">`, "")
		cleanResponse = strings.ReplaceAll(cleanResponse, `</span>`, "")

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
