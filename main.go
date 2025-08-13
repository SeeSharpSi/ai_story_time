package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"story_ai/handlers"
	"story_ai/session"
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
	sessionManager := session.NewManager()

	h := &handlers.Handler{
		Model:   model,
		Manager: sessionManager,
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		templates.Index("Interactive Story").Render(context.Background(), w)
	})

	mux.HandleFunc("/start", h.StartStory)
	mux.HandleFunc("/generate", h.Generate)
	mux.HandleFunc("/download", h.DownloadStory)

	log.Println("Listening on http://0.0.0.0:9779")
	log.Fatal(http.ListenAndServe("0.0.0.0:9779", mux))
}