package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
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
	authors = []string{"James Joyce", "Mark Twain", "Jack Kerouac", "Kurt Vonnegut", "H.P. Lovecraft", "Edgar Allan Poe", "J.R.R. Tolkien", "Terry Pratchett"}
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

	if s.IsFunny {
		prompt += prompts.FunnyStoryPrompt
	} else if s.IsAngry {
		prompt += prompts.AngryPrompt
	} else if s.IsXKCD {
		prompt += prompts.XKCDPrompt
	} else if s.IsStanley {
		prompt += prompts.StanleyPrompt
	} else if s.IsGLaDOS {
		prompt += prompts.GLaDOSPrompt
	} else if s.IsKreia {
		prompt += prompts.KreiaPrompt
	} else if s.IsNietzsche {
		prompt += prompts.NietzschePrompt
	} else if s.IsJohnBunyan {
		prompt += prompts.BunyanPrompt
	} else if s.IsSocrates {
		prompt += prompts.SocraticPrompt
	} else if s.IsTheHistorian {
		prompt += prompts.HistorianPrompt
	} else if s.IsRossRamsay {
		prompt += prompts.RossRamsayPrompt
	} else if s.IsTzuGump {
		prompt += prompts.SunTzuGumpPrompt
	} else if s.IsDrSeuss {
		prompt += prompts.DrSeussPrompt
	}

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

func (h *Handler) StartStory(w http.ResponseWriter, r *http.Request) {
	sess, cookie := h.Manager.GetOrCreateSession(r)
	http.SetCookie(w, &cookie)

	genre := r.URL.Query().Get("genre")
	consequenceModel := r.URL.Query().Get("consequence_model")
	sess.GameState.Rules.ConsequenceModel = consequenceModel
	sess.CurrentGenre = genre

	// Reset story history for a new game
	sess.StoryHistory = []story.StoryPage{}
	sess.IsFunny = false        // Reset flag for new stories
	sess.IsAngry = false        // Reset flag for new stories
	sess.IsXKCD = false         // Reset flag for new stories
	sess.IsStanley = false      // Reset flag for new stories
	sess.IsGLaDOS = false       // Reset flag for new stories
	sess.IsKreia = false        // Reset flag for new stories
	sess.IsNietzsche = false    // Reset flag for new stories
	sess.IsJohnBunyan = false   // Reset flag for new stories
	sess.IsSocrates = false     // Reset flag for new stories
	sess.IsTheHistorian = false // Reset flag for new stories
	sess.IsRossRamsay = false   // Reset flag for new stories
	sess.IsTzuGump = false      // Reset flag for new stories
	sess.IsDrSeuss = false      // Reset flag for new stories

	author := ""
	narrative_dice := rand.Intn(13) // Roll a number from 0 to 12

	switch {
	case narrative_dice < 1:
		sess.IsAngry = true
		author = "a very angry narrator"

	case narrative_dice < 2 && genre != "historical-fiction":
		sess.IsFunny = true
		author = "the Monty Python group"

	// This style is exclusive to the sci-fi genre
	case narrative_dice < 3 && genre == "sci-fi":
		sess.IsXKCD = true
		author = "XKCD"

	case narrative_dice < 4:
		sess.IsStanley = true
		author = "The Stanley Parable"

	// This style is exclusive to the sci-fi genre
	case narrative_dice < 5 && genre == "sci-fi":
		sess.IsGLaDOS = true
		author = "GLaDOS from Portal 2"

	// This style is exclusive to the fantasy genre
	case narrative_dice < 5 && genre == "fantasy":
		sess.IsKreia = true
		author = "Kreia from Knights of the Old Republic II"

	// This style is exclusive to the historical fiction genre
	case narrative_dice < 5 && genre == "historical-fiction":
		sess.IsTheHistorian = true
		author = "The Historian"

	case narrative_dice < 6:
		sess.IsNietzsche = true
		author = "Friedrich Nietzsche"

	case narrative_dice < 7:
		sess.IsJohnBunyan = true
		author = "John Bunyan"

	case narrative_dice < 8:
		sess.IsSocrates = true
		author = "Socrates"

	case narrative_dice < 9:
		sess.IsRossRamsay = true
		author = "Ross & Ramsay"

	case narrative_dice < 10:
		sess.IsTzuGump = true
		author = "Sun Tzu & Forrest Gump"

	case narrative_dice < 11 && genre != "historical-fiction":
		sess.IsDrSeuss = true
		author = "Dr. Seuss"

	default:
		author = authors[rand.Intn(len(authors))]
	}

	sess.CurrentAuthor = author
	log.Printf("--- NEW STORY --- Author: %s, Genre: %s, Difficulty: %s", author, genre, consequenceModel)
	go pingStatsService("start", nil)

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

	log.Printf("--- NEW GAME STATE (START) --- %s", prettyPrint(aiResp.NewGameState))

	if aiResp.StoryUpdate.BackgroundColor == "" {
		aiResp.StoryUpdate.BackgroundColor = "#1e1e1e"
	}

	sess.GameState = aiResp.NewGameState
	// The FoundItems list will be empty on start, so no need to update it yet.
	storyText := aiResp.StoryUpdate.Story
	if sess.IsStanley && !strings.HasPrefix(storyText, "This is the story of a man named Stanley.") {
		storyText = "This is the story of a man named Stanley.<br><br>" + storyText
	}
	sess.StoryHistory = []story.StoryPage{{Prompt: "Start", Response: storyText}}

	placeholder := "What do you do?"
	if sess.IsStanley {
		placeholder = "What does Stanley do?"
	} else if sess.IsDrSeuss {
		placeholder = "Your turn to play! What's next today?"
	}
	templates.StoryView(storyText, aiResp.NewGameState.PlayerStatus, aiResp.NewGameState.Inventory, aiResp.StoryUpdate.BackgroundColor, genre, aiResp.NewGameState.World.WorldTension, consequenceModel, placeholder).Render(context.Background(), w)
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

	log.Printf("--- NEW GAME STATE (GENERATE) --- %s", prettyPrint(aiResp.NewGameState))

	if aiResp.StoryUpdate.BackgroundColor == "" {
		aiResp.StoryUpdate.BackgroundColor = "#1e1e1e"
	}

	if aiResp.StoryUpdate.GameOver || aiResp.NewGameState.GameWon {
		go pingStatsService("complete", nil)
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

	storyText := aiResp.StoryUpdate.Story // Use nouns from this turn for tooltips
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

			if goquery.NodeName(content) == "br" {
				pdf.Ln(6)
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
	title := "Your Story"
	if sess.IsFunny {
		title = "A Decently Amusing Story"
	} else if sess.IsAngry {
		title = "The Tale I Was Forced to Tell"
	} else if sess.IsXKCD {
		title = "Hypothesis: A Story"
	} else if sess.IsStanley {
		title = "The Story of a Man Named Stanley"
	} else if sess.IsGLaDOS {
		title = "A Mandatory Enrichment Activity"
	} else if sess.IsKreia {
		title = "A Lesson in Consequences"
	} else if sess.IsNietzsche {
		title = "Thus Spoke the Traveler"
	} else if sess.IsJohnBunyan {
		title = "The Pilgrim's Burden"
	} else if sess.IsSocrates {
		title = "An Unexamined Life"
	} else if sess.IsTheHistorian {
		title = "The Human Thing"
	} else if sess.IsRossRamsay {
		title = "The Happy Little Scallop is RAW!"
	} else if sess.IsTzuGump {
		title = "The Unwitting Strategist"
	} else if sess.IsDrSeuss {
		title = "Oh, the Things You Will Find!"
	}

	pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")
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

	var pdfBuffer bytes.Buffer
	err := pdf.Output(&pdfBuffer)
	if err != nil {
		log.Printf("Error generating PDF to buffer: %v", err)
		http.Error(w, "Failed to generate PDF.", http.StatusInternalServerError)
		return
	}

	go pingStatsService("upload-pdf", pdfBuffer.Bytes())

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=story.pdf")
	w.Write(pdfBuffer.Bytes())
}

// pingStatsService sends a POST request to the stats service.
// For sending PDFs, the pdfData should contain the raw PDF bytes.
func pingStatsService(endpoint string, pdfData []byte) {
	statsServiceURL := os.Getenv("STATS_SERVICE_URL") // Get URL from environment variable
	if statsServiceURL == "" {
		// You can set a default for local testing
		statsServiceURL = "http://localhost:8080"
	}

	url := fmt.Sprintf("%s/%s", statsServiceURL, endpoint)

	var req *http.Request
	var err error

	if pdfData != nil {
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(pdfData))
		req.Header.Set("Content-Type", "application/pdf")
	} else {
		req, err = http.NewRequest("POST", url, nil)
	}

	if err != nil {
		log.Printf("Error creating request to stats service: %v", err)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error pinging stats service at '%s': %v", endpoint, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Stats service returned non-OK status for '%s': %s", endpoint, resp.Status)
	} else {
		log.Printf("Successfully pinged stats service at '%s'", endpoint)
	}
}
