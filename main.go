package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"story_ai/handlers"
	"story_ai/templates"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")

	h := &handlers.Handler{
		Model: model,
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// The root now renders the welcome page.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		templates.Index("Interactive Story").Render(context.Background(), w)
	})

	// The /start route begins the story.
	mux.HandleFunc("/start", h.StartStory)
	mux.HandleFunc("/generate", h.Generate)

	log.Println("Listening on :9779")
	log.Fatal(http.ListenAndServe(":9779", mux))
}
