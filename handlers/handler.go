package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// AIResponse is the top-level structure for the AI's JSON response.
type AIResponse struct {
	NewGameState *story.GameState `json:"new_game_state"`
	StoryUpdate  StoryUpdate      `json:"story_update"`
}

// StoryUpdate contains the narrative portion of the AI's response.
type StoryUpdate struct {
	Story           string   `json:"story"`
	ItemsAdded      []string `json:"items_added"`
	ItemsRemoved    []string `json:"items_removed"`
	GameOver        bool     `json:"game_over"`
	BackgroundColor string   `json:"background_color"`
}

// AIRequest is the structure sent to the AI.
type AIRequest struct {
	GameState  *story.GameState `json:"game_state"`
	UserAction string           `json:"user_action"`
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

	aiResp.StoryUpdate.Story = markdownBoldRegex.ReplaceAllString(aiResp.StoryUpdate.Story, "<strong>$1</strong>")
	aiResp.StoryUpdate.Story = markdownItalicRegex.ReplaceAllString(aiResp.StoryUpdate.Story, "<em>$1</em>")

	return aiResp, nil
}

func prettyPrint(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error pretty printing: %v", err)
	}
	return string(b)
}

func (h *Handler) StartStory(w http.ResponseWriter, r *http.Request) {
	sess, cookie := h.Manager.GetOrCreateSession(r)
	http.SetCookie(w, &cookie)

	genre := r.URL.Query().Get("genre")
	consequenceModel := r.URL.Query().Get("consequence_model")
	sess.GameState.Rules.ConsequenceModel = consequenceModel

	// Reset story history for a new game
	sess.StoryHistory = []story.StoryPage{}

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
	prompt = fmt.Sprintf(prompts.BasePrompt, sess.CurrentAuthor)

	switch genre {
	case "fantasy":
		prompt += prompts.FantasyPrompt
		sess.CurrentGenre = "fantasy"
	case "sci-fi":
		prompt += prompts.SciFiPrompt
		sess.CurrentGenre = "sci-fi"
	case "historical-fiction":
		prompt += prompts.HistoricalFictionPrompt
		sess.CurrentGenre = "historical-fiction"
	default:
		prompt += prompts.FantasyPrompt
		sess.CurrentGenre = "fantasy"
	}

	// The initial game state is empty, the AI will generate the starting scenario.
	initialRequest := AIRequest{
		GameState:  &story.GameState{Rules: story.Rules{ConsequenceModel: consequenceModel}},
		UserAction: "Start the game.",
	}
	reqBytes, err := json.Marshal(initialRequest)
	if err != nil {
		http.Error(w, "Failed to create initial AI request.", http.StatusInternalServerError)
		return
	}

	fullPrompt := prompt + string(reqBytes)

	resp, err := h.Model.GenerateContent(context.Background(), genai.Text(fullPrompt))
	if err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		http.Error(w, "The AI failed to start the story. Please try again.", http.StatusInternalServerError)
		return
	}

	aiResp, err := parseAIResponse(string(resp.Candidates[0].Content.Parts[0].(genai.Text)))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse AI's initial response: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("--- NEW GAME STATE (START) --- %s", prettyPrint(aiResp.NewGameState))

	if aiResp.StoryUpdate.BackgroundColor == "" {
		aiResp.StoryUpdate.BackgroundColor = "#1e1e1e"
	}

	sess.GameState = aiResp.NewGameState
	sess.StoryHistory = []story.StoryPage{{Prompt: "Start", Response: aiResp.StoryUpdate.Story}}

	templates.StoryView(aiResp.StoryUpdate.Story, aiResp.NewGameState.PlayerStatus, aiResp.NewGameState.Inventory, aiResp.StoryUpdate.BackgroundColor).Render(context.Background(), w)
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Manager.GetOrCreateSession(r)
	userAction := r.FormValue("prompt")

	if strings.ToLower(strings.TrimSpace(userAction)) == "restart" {
		query := "genre=" + sess.CurrentGenre + "&consequence_model=" + sess.GameState.Rules.ConsequenceModel
		r.URL.RawQuery = query
		h.StartStory(w, r)
		return
	}

	if len(strings.Fields(userAction)) > 15 {
		http.Error(w, "Response must be 15 words or less.", http.StatusBadRequest)
		return
	}

	var systemPrompt string
	systemPrompt = fmt.Sprintf(prompts.BasePrompt, sess.CurrentAuthor)

	switch sess.CurrentGenre {
	case "fantasy":
		systemPrompt += prompts.FantasyPrompt
	case "sci-fi":
		systemPrompt += prompts.SciFiPrompt
	case "historical-fiction":
		systemPrompt += prompts.HistoricalFictionPrompt
	default:
		systemPrompt += prompts.FantasyPrompt
	}

	aiRequest := AIRequest{
		GameState:  sess.GameState,
		UserAction: userAction,
	}
	reqBytes, err := json.Marshal(aiRequest)
	if err != nil {
		http.Error(w, "Failed to create AI request.", http.StatusInternalServerError)
		return
	}

	fullPrompt := systemPrompt + string(reqBytes)

	resp, err := h.Model.GenerateContent(r.Context(), genai.Text(fullPrompt))
	if err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		errorPage := story.StoryPage{Prompt: userAction, Response: "[The AI's response was blocked. Try something else.]"}
		sess.StoryHistory = append(sess.StoryHistory, errorPage)
		templates.Update(sess.StoryHistory, sess.GameState.PlayerStatus, sess.GameState.Inventory, "#1e1e1e", false, sess.CurrentGenre, sess.GameState.Rules.ConsequenceModel).Render(context.Background(), w)
		return
	}

	aiResp, err := parseAIResponse(string(resp.Candidates[0].Content.Parts[0].(genai.Text)))
	if err != nil {
		errorPage := story.StoryPage{Prompt: userAction, Response: fmt.Sprintf("[The AI's response was not valid JSON: %v]", err)}
		sess.StoryHistory = append(sess.StoryHistory, errorPage)
		templates.Update(sess.StoryHistory, sess.GameState.PlayerStatus, sess.GameState.Inventory, "#1e1e1e", false, sess.CurrentGenre, sess.GameState.Rules.ConsequenceModel).Render(context.Background(), w)
		return
	}

	log.Printf("--- NEW GAME STATE (GENERATE) --- %s", prettyPrint(aiResp.NewGameState))

	if aiResp.StoryUpdate.BackgroundColor == "" {
		aiResp.StoryUpdate.BackgroundColor = "#1e1e1e"
	}

	sess.GameState = aiResp.NewGameState
	sess.StoryHistory = append(sess.StoryHistory, story.StoryPage{Prompt: userAction, Response: aiResp.StoryUpdate.Story})
	templates.Update(sess.StoryHistory, sess.GameState.PlayerStatus, sess.GameState.Inventory, aiResp.StoryUpdate.BackgroundColor, aiResp.StoryUpdate.GameOver, sess.CurrentGenre, sess.GameState.Rules.ConsequenceModel).Render(context.Background(), w)
}

func (h *Handler) DownloadStory(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Manager.GetOrCreateSession(r)
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 24)
	pdf.Cell(0, 10, "Your Story")
	pdf.Ln(15)

	pdf.SetFont("Helvetica", "I", 14)
	pdf.Cell(0, 10, "An AI-generated story in the style of "+sess.CurrentAuthor+" set in a world of "+sess.CurrentGenre)
	pdf.Ln(20)

	pdf.SetFont("Helvetica", "", 12)
	for _, page := range sess.StoryHistory {
		pdf.SetFontStyle("I")
		pdf.Write(5, page.Prompt)
		pdf.Ln(10)

		pdf.SetFontStyle("")
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