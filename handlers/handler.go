package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"story_ai/prompts"
	"story_ai/session"
	"story_ai/story"
	"story_ai/templates"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/generative-ai-go/genai"
	"github.com/jung-kurt/gofpdf"
	_ "modernc.org/sqlite"
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
	authors = []string{"James Joyce", "Mark Twain", "Jack Kerouac", "Kurt Vonnegut", "H.P. Lovecraft", "Edgar Allan Poe", "J.R.R. Tolkien", "Terry Pratchett", "Other"}
	// Regex to find Markdown bolding (**text**)
	markdownBoldRegex = regexp.MustCompile(`\*\*(.*?)\*\*`)
	// Regex to find Markdown italics (*text*)
	markdownItalicRegex = regexp.MustCompile(`\*(.*?)\*`)
)

// addTooltipSpans finds all occurrences of proper nouns from the game state in the
// story text and wraps them in HTML spans for tooltips. It uses a two-pass
// placeholder system to prevent nesting tooltips within each other.
func addTooltipSpans(storyText string, nouns []story.ProperNoun) string {
	// Sort nouns by length in descending order to match longer phrases first.
	sort.Slice(nouns, func(i, j int) bool {
		return len(nouns[i].PhraseUsed) > len(nouns[j].PhraseUsed)
	})

	placeholderMap := make(map[string]story.ProperNoun)
	currentStory := storyText
	placeholderIndex := 0

	// Pass 1: Replace all phrases with unique, safe placeholders.
	for _, noun := range nouns {
		re := regexp.MustCompile(fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(noun.PhraseUsed)))
		
		// Use ReplaceAllStringFunc to avoid replacing parts of already-made placeholders
		// or HTML tags from previous replacements in this loop.
		currentStory = re.ReplaceAllStringFunc(currentStory, func(match string) string {
			// Check if we are about to replace something that is already a placeholder
			if strings.HasPrefix(match, "__NOUN_PLACEHOLDER_") {
				return match
			}
			
			placeholder := fmt.Sprintf("__NOUN_PLACEHOLDER_%d__", placeholderIndex)
			placeholderMap[placeholder] = noun
			placeholderIndex++
			return placeholder
		})
	}

	// Pass 2: Replace placeholders with the final HTML.
	finalStory := currentStory
	for placeholder, noun := range placeholderMap {
		wrapper := fmt.Sprintf(`<span class="proper-noun tooltip">%s<span class="tooltiptext">%s</span></span>`, noun.PhraseUsed, noun.Description)
		finalStory = strings.Replace(finalStory, placeholder, wrapper, -1)
	}

	return finalStory
}

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

func (h *Handler) parseAndRetryAIResponse(ctx context.Context, originalResponse string) (AIResponse, error) {
	aiResp, err := parseAIResponse(originalResponse)
	if err == nil {
		return aiResp, nil
	}

	log.Printf("Initial JSON parsing failed: %v. Retrying with the AI.", err)

	for i := 0; i < 3; i++ { // Retry up to 3 times
		retryPrompt := fmt.Sprintf(prompts.JsonRetryPrompt, originalResponse)
		resp, retryErr := h.Model.GenerateContent(ctx, genai.Text(retryPrompt))
		if retryErr != nil {
			log.Printf("AI retry attempt %d failed: %v", i+1, retryErr)
			continue
		}

		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			correctedResponse := string(resp.Candidates[0].Content.Parts[0].(genai.Text))
			aiResp, err = parseAIResponse(correctedResponse)
			if err == nil {
				log.Printf("AI successfully corrected the JSON on attempt %d.", i+1)
				return aiResp, nil
			}
			log.Printf("AI retry attempt %d still resulted in invalid JSON: %v", i+1, err)
			originalResponse = correctedResponse // Use the corrected (but still invalid) response for the next retry
		}
	}

	return AIResponse{}, fmt.Errorf("failed to parse AI response after multiple retries")
}

func prettyPrint(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error pretty printing: %v", err)
	}
	return string(b)
}

func (h *Handler) buildSystemPrompt(s *session.Session) string {
	prompt := fmt.Sprintf(prompts.BasePrompt, s.CurrentAuthor)

	switch s.CurrentGenre {
	case "fantasy":
		prompt += prompts.FantasyPrompt
	case "sci-fi":
		prompt += prompts.SciFiPrompt
	case "historical-fiction":
		prompt += fmt.Sprintf(prompts.HistoricalFictionPrompt, s.HistoricalEvent, s.HistoricalDesc, s.HistoricalSummary)
	default:
		prompt += prompts.FantasyPrompt
	}
	return prompt
}

// validateAndCorrectProperNouns checks if the phrase_used for a proper noun is in the story.
// If not, it tries to correct it by using the canonical noun. If neither is found,
// the proper noun is removed from the list to prevent broken tooltips.
func validateAndCorrectProperNouns(aiResp *AIResponse) {
	if aiResp.NewGameState == nil {
		return
	}

	var validatedNouns []story.ProperNoun
	storyTextLower := strings.ToLower(aiResp.StoryUpdate.Story)

	for _, noun := range aiResp.NewGameState.ProperNouns {
		phraseUsedLower := strings.ToLower(noun.PhraseUsed)
		nounLower := strings.ToLower(noun.Noun)

		// To be valid, the phrase must exist as a whole word or phrase in the story.
		// We construct a regex to check for this. \b is a word boundary.
		phraseRegex := regexp.MustCompile(`\b` + regexp.QuoteMeta(phraseUsedLower) + `\b`)
		nounRegex := regexp.MustCompile(`\b` + regexp.QuoteMeta(nounLower) + `\b`)

		if phraseRegex.MatchString(storyTextLower) {
			// The phrase_used is in the story, so it's valid.
			validatedNouns = append(validatedNouns, noun)
		} else if nounRegex.MatchString(storyTextLower) {
			// The phrase_used was incorrect, but the canonical noun is in the story.
			// We'll correct the phrase_used and keep the noun.
			log.Printf("Correcting proper noun: phrase_used '%s' not found, but noun '%s' was. Updating phrase_used.", noun.PhraseUsed, noun.Noun)
			noun.PhraseUsed = noun.Noun // Correct the phrase to the canonical noun.
			validatedNouns = append(validatedNouns, noun)
		} else {
			// Neither the phrase_used nor the noun is in the story. Discard it.
			log.Printf("Discarding proper noun: phrase_used '%s' and noun '%s' not found in story.", noun.PhraseUsed, noun.Noun)
		}
	}
	aiResp.NewGameState.ProperNouns = validatedNouns
}

func (h *Handler) StartStory(w http.ResponseWriter, r *http.Request) {
	sess, cookie := h.Manager.GetOrCreateSession(r)
	http.SetCookie(w, &cookie)


	genre := r.URL.Query().Get("genre")
	consequenceModel := r.URL.Query().Get("consequence_model")
	sess.GameState.Rules.ConsequenceModel = consequenceModel
	sess.CurrentGenre = genre

	// Reset story history for a new game
	sess.StoryHistory = []story.StoryPage{}

	rand.Seed(time.Now().UnixNano())
	author := authors[rand.Intn(len(authors))]

	if author == "Other" {
		authorPrompt := "Name one famous author who is not on this list: William Faulkner, James Joyce, Mark Twain, Jack Kerouac, Kurt Vonnegut, H.P. Lovecraft, Edgar Allan Poe, J.R.R. Tolkien, Douglas Adams, Terry Pratchett. Respond with only the author's name."
		resp, err := h.Model.GenerateContent(context.Background(), genai.Text(authorPrompt))
		if err != nil || len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			author = "Mark Twain"
		} else {
			author = string(resp.Candidates[0].Content.Parts[0].(genai.Text))
		}
	}
	sess.CurrentAuthor = author
	log.Printf("--- NEW STORY --- Author: %s, Genre: %s, Difficulty: %s", author, genre, consequenceModel)

	var prompt string
	var inspirationTitle, inspirationDesc string

	db, err := sql.Open("sqlite", "./data.db")
	if err != nil {
		http.Error(w, "Failed to open database.", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	switch genre {
	case "fantasy":
		err = db.QueryRow("SELECT title, description FROM fantasy_inspo ORDER BY RANDOM() LIMIT 1").Scan(&inspirationTitle, &inspirationDesc)
		if err != nil {
			http.Error(w, "Failed to query database for fantasy inspiration.", http.StatusInternalServerError)
			return
		}
		prompt = h.buildSystemPrompt(sess)
		prompt += fmt.Sprintf("\n- You MUST use the following title and description as inspiration for the story:\n- Title: %s\n- Description: %s\n", inspirationTitle, inspirationDesc)
	case "sci-fi":
		err = db.QueryRow("SELECT title, description FROM scifi_inspo ORDER BY RANDOM() LIMIT 1").Scan(&inspirationTitle, &inspirationDesc)
		if err != nil {
			http.Error(w, "Failed to query database for sci-fi inspiration.", http.StatusInternalServerError)
			return
		}
		prompt = h.buildSystemPrompt(sess)
		prompt += fmt.Sprintf("\n- You MUST use the following title and description as inspiration for the story:\n- Title: %s\n- Description: %s\n", inspirationTitle, inspirationDesc)
	case "historical-fiction":
		var wikipediaURL string
		err = db.QueryRow("SELECT event, description, wikipedia, summary FROM historical_events ORDER BY RANDOM() LIMIT 1").Scan(&sess.HistoricalEvent, &sess.HistoricalDesc, &wikipediaURL, &sess.HistoricalSummary)
		if err != nil {
			http.Error(w, "Failed to query database for historical event.", http.StatusInternalServerError)
			return
		}
		sess.HistoricalURL = wikipediaURL
		prompt = h.buildSystemPrompt(sess)
		log.Printf("--- HISTORICAL EVENT --- Event: %s, Description: %s", sess.HistoricalEvent, sess.HistoricalDesc)
	default:
		sess.CurrentGenre = "fantasy" // Default to fantasy
		prompt = h.buildSystemPrompt(sess)
	}

	// The initial game state is empty, the AI will generate the starting scenario.
	initialRequest := AIRequest{
		GameState: &story.GameState{
			Rules:       story.Rules{ConsequenceModel: consequenceModel},
			World:       story.World{WorldTension: 0},
			Climax:      false,
			ProperNouns: []story.ProperNoun{},
		},
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

	aiResp, err := h.parseAndRetryAIResponse(context.Background(), string(resp.Candidates[0].Content.Parts[0].(genai.Text)))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse AI's initial response: %v", err), http.StatusInternalServerError)
		return
	}

	validateAndCorrectProperNouns(&aiResp)

	log.Printf("--- NEW GAME STATE (START) --- %s", prettyPrint(aiResp.NewGameState))

	if aiResp.StoryUpdate.BackgroundColor == "" {
		aiResp.StoryUpdate.BackgroundColor = "#1e1e1e"
	}

	sess.GameState = aiResp.NewGameState
	// The FoundItems list will be empty on start, so no need to update it yet.
	storyText := addTooltipSpans(aiResp.StoryUpdate.Story, sess.GameState.ProperNouns)
	sess.StoryHistory = []story.StoryPage{{Prompt: "Start", Response: storyText}}

	templates.StoryView(storyText, aiResp.NewGameState.PlayerStatus, aiResp.NewGameState.Inventory, aiResp.StoryUpdate.BackgroundColor, genre, aiResp.NewGameState.World.WorldTension, consequenceModel).Render(context.Background(), w)
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

	systemPrompt := h.buildSystemPrompt(sess)

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
		templates.Update(sess.StoryHistory, sess.GameState.PlayerStatus, sess.GameState.Inventory, "#1e1e1e", false, false, sess.CurrentGenre, sess.GameState.Rules.ConsequenceModel, sess.GameState.World.WorldTension).Render(context.Background(), w)
		return
	}

	aiResp, err := h.parseAndRetryAIResponse(r.Context(), string(resp.Candidates[0].Content.Parts[0].(genai.Text)))
	if err != nil {
		errorPage := story.StoryPage{Prompt: userAction, Response: fmt.Sprintf("[The AI's response was not valid JSON: %v]", err)}
		sess.StoryHistory = append(sess.StoryHistory, errorPage)
		templates.Update(sess.StoryHistory, sess.GameState.PlayerStatus, sess.GameState.Inventory, "#1e1e1e", false, false, sess.CurrentGenre, sess.GameState.Rules.ConsequenceModel, sess.GameState.World.WorldTension).Render(context.Background(), w)
		return
	}

	validateAndCorrectProperNouns(&aiResp)

	log.Printf("--- NEW GAME STATE (GENERATE) --- %s", prettyPrint(aiResp.NewGameState))

	if aiResp.StoryUpdate.BackgroundColor == "" {
		aiResp.StoryUpdate.BackgroundColor = "#1e1e1e"
	}

	// Merge the AI's proper nouns into the session's master list.
	existingNouns := make(map[string]bool)
	for _, noun := range sess.GameState.ProperNouns {
		existingNouns[noun.Noun] = true
	}
	for _, newNoun := range aiResp.NewGameState.ProperNouns {
		if !existingNouns[newNoun.Noun] {
			sess.GameState.ProperNouns = append(sess.GameState.ProperNouns, newNoun)
			existingNouns[newNoun.Noun] = true
		}
	}

	// Update the rest of the game state, but preserve our master noun list.
	updatedNouns := sess.GameState.ProperNouns
	sess.GameState = aiResp.NewGameState
	sess.GameState.ProperNouns = updatedNouns

	storyText := addTooltipSpans(aiResp.StoryUpdate.Story, aiResp.NewGameState.ProperNouns) // Use nouns from this turn for tooltips
	sess.StoryHistory = append(sess.StoryHistory, story.StoryPage{Prompt: userAction, Response: storyText})
	templates.Update(sess.StoryHistory, sess.GameState.PlayerStatus, sess.GameState.Inventory, aiResp.StoryUpdate.BackgroundColor, aiResp.StoryUpdate.GameOver, sess.GameState.GameWon, sess.CurrentGenre, sess.GameState.Rules.ConsequenceModel, sess.GameState.World.WorldTension).Render(context.Background(), w)
}

// writeHtmlToPdf parses a simple HTML string and writes it to the PDF, handling nested styles.
func writeHtmlToPdf(pdf *gofpdf.Fpdf, htmlStr string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<body>" + htmlStr + "</body>"))
	if err != nil {
		pdf.MultiCell(0, 6, htmlStr, "", "", false)
		return
	}

	var currentStyle string
	var process func(*goquery.Selection)
	process = func(s *goquery.Selection) {
		s.Contents().Each(func(i int, content *goquery.Selection) {
			// Skip tooltiptext spans entirely in the main processing loop
			if content.HasClass("tooltiptext") {
				return
			}

			if goquery.NodeName(content) == "#text" {
				pdf.SetFontStyle(currentStyle)
				pdf.Write(6, content.Text())
			} else {
				// Store parent's state
				r, g, b := pdf.GetTextColor()
				parentStyle := currentStyle

				// Determine new style for this node's children
				switch goquery.NodeName(content) {
				case "strong":
					if !strings.Contains(currentStyle, "B") {
						currentStyle += "B"
					}
				case "em":
					if !strings.Contains(currentStyle, "I") {
						currentStyle += "I"
					}
				case "span":
					if content.HasClass("item-added") {
						pdf.SetTextColor(34, 139, 34) // ForestGreen
					} else if content.HasClass("item-removed") {
						pdf.SetTextColor(165, 42, 42) // Brown
						if !strings.Contains(currentStyle, "S") {
							currentStyle += "S"
						}
					} else if content.HasClass("proper-noun") {
						// Proper nouns are bolded in the PDF instead of colored
						if !strings.Contains(currentStyle, "B") {
							currentStyle += "B"
						}
					}
				}

				// Process children with the new style
				process(content)

				// Restore parent's state for subsequent siblings
				pdf.SetTextColor(r, g, b)
				currentStyle = parentStyle
				pdf.SetFontStyle(currentStyle)
			}
		})
	}
	process(doc.Find("body"))
	pdf.SetFontStyle("") // Final reset
}

func (h *Handler) DownloadStory(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Manager.GetOrCreateSession(r)
	pdf := gofpdf.New("P", "mm", "A4", "")

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Times", "I", 8)
		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "C", false, 0, "")
	})

	// Title Page
	pdf.AddPage()
	pdf.SetFont("Times", "B", 36)
	pdf.Cell(0, 80, "")
	pdf.Ln(-1)
	pdf.CellFormat(0, 10, "Your Story", "", 1, "C", false, 0, "")
	pdf.Ln(10)

	pdf.SetFont("Times", "I", 16)
	subtitle := fmt.Sprintf("An AI-generated %s tale in the style of %s",
		strings.Title(sess.CurrentGenre),
		sess.CurrentAuthor,
	)
	pdf.CellFormat(0, 10, subtitle, "", 1, "C", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Times", "", 12)
	difficulty := fmt.Sprintf("Difficulty: %s", strings.Title(sess.GameState.Rules.ConsequenceModel))
	pdf.CellFormat(0, 10, difficulty, "", 1, "C", false, 0, "")

	if sess.CurrentGenre == "historical-fiction" {
		pdf.Ln(20)
		pdf.SetFont("Times", "B", 14)
		pdf.CellFormat(0, 10, "Historical Context", "", 1, "C", false, 0, "")
		pdf.Ln(5)
		pdf.SetFont("Times", "", 12)
		contextText := fmt.Sprintf("Event: %s\n%s", sess.HistoricalEvent, sess.HistoricalDesc)
		pdf.MultiCell(0, 6, contextText, "", "C", false)
		pdf.Ln(2)
		pdf.SetFont("Times", "I", 10)
		pdf.SetTextColor(65, 105, 225) // RoyalBlue
		pdf.CellFormat(0, 10, sess.HistoricalURL, "", 1, "C", false, 0, sess.HistoricalURL)
		pdf.SetTextColor(0, 0, 0)
	}

	// Story Pages
	pdf.AddPage()
	pdf.SetFont("Times", "", 12)
	pdf.SetMargins(20, 20, 20)

	for _, page := range sess.StoryHistory {
		pdf.SetFont("Times", "I", 12)
		pdf.SetTextColor(64, 64, 64)
		pdf.MultiCell(0, 6, "> "+page.Prompt, "", "", false)
		pdf.Ln(6)

		pdf.SetFont("Times", "", 12)
		pdf.SetTextColor(0, 0, 0)
		writeHtmlToPdf(pdf, page.Response)
		pdf.Ln(12)
	}

	// Glossary Page
	if len(sess.GameState.ProperNouns) > 0 {
		pdf.AddPage()
		pdf.SetFont("Times", "B", 24)
		pdf.CellFormat(0, 10, "Glossary of Terms", "", 1, "C", false, 0, "")
		pdf.Ln(10)
		pdf.SetFont("Times", "", 12)
		for _, noun := range sess.GameState.ProperNouns {
			pdf.SetFont("Times", "B", 12)
			pdf.Write(6, noun.Noun+": ")
			pdf.SetFont("Times", "", 12)
			pdf.Write(6, noun.Description)
			pdf.Ln(8)
		}
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=story.pdf")
	err := pdf.Output(w)
	if err != nil {
		log.Printf("Error generating PDF: %v", err)
		http.Error(w, "Failed to generate PDF.", http.StatusInternalServerError)
	}
}
