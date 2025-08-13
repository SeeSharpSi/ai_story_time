# Story AI 🤖✍️

**Story AI is an interactive, text-based adventure game powered by Google's Gemini API.** Players craft a unique story by responding to AI-generated scenarios, with their choices directly shaping the narrative, the items they discover, and the world they explore.

---

![Mark Twain writing a sci-fi book.](https://drive.google.com/file/d/1y1aEXO-485oiPrbnfNjZMNsbsR1KMZu5/view?usp=sharing)

---

## ✨ Features

*   **Dynamic AI Storytelling:** Powered by Google's Gemini model, every adventure is unique and unpredictable. The story adapts to your choices in real-time.
*   **Author-Styled Narratives:** Begin your adventure in the literary style of a famous author, like William Faulkner or Mark Twain, for a unique narrative flavor.
*   **Hardcore Survival Mode:** Flip the "Survive" switch to raise the stakes. In this mode, the AI prioritizes challenging, dangerous outcomes where poor choices can easily lead to your demise.
*   **Genre Selection:** Choose from Fantasy, Sci-Fi, or Historical Fiction to guide the theme of your adventure.
*   **Interactive Inventory:** The AI tracks items you pick up or lose, which you can use to solve problems and navigate the world.
*   **Download Your Story:** Once your adventure concludes, you can download the entire story as a beautifully formatted PDF to save or share.
*   **Modern, Fast Frontend:** The UI is built with Go, HTMX, and Templ, delivering a seamless, server-rendered experience without heavy client-side JavaScript.

## 🛠️ Tech Stack

*   **Backend:** Go
*   **Frontend:** HTMX & [a-h/templ](https://github.com/a-h/templ)
*   **AI Model:** Google Gemini
*   **PDF Generation:** [gofpdf](https://github.com/jung-kurt/gofpdf)

## 🚀 Getting Started

Follow these steps to get Story AI running on your local machine.

### 1. Prerequisites

*   Go 1.21 or later.
*   A Google Gemini API key. You can get one for free from [Google AI Studio](https://aistudio.google.com/app/apikey).

### 2. Clone the Repository

```bash
git clone https://github.com/your-username/story_ai
cd story_ai
```

### 3. Set Up Your Environment

Create a `.env` file in the root of the project directory. This file will hold your Gemini API key.

```
GEMINI_API_KEY=YOUR_API_KEY
```

Replace `YOUR_API_KEY` with your actual Gemini API key.

### 4. Run the Application

First, ensure the templ files are generated:
```bash
templ generate
```

Then, run the application:
```bash
go run .
```

The application will be available at `http://localhost:9779`.

## 🎮 How to Play

1.  Open your web browser to `http://localhost:9779`.
2.  **Choose Your Adventure:** Select a genre (Fantasy, Sci-Fi, or Historical Fiction).
3.  **(Optional) Flip the Switch:** For a true challenge, enable the "Survive" mode.
4.  Click a genre button to begin!
5.  Read the AI-generated scenario and type your response (15 words or less) into the input box.
6.  Click "Send" and watch the story unfold based on your choices.
7.  If your story reaches a conclusion, you can restart or download your adventure as a PDF. Good luck!
