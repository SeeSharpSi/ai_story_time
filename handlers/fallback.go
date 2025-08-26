package handlers

import (
	"fmt"
	"math/rand"
	"story_ai/story"
	"time"
)

// FallbackStoryGenerator provides basic story generation when AI is unavailable
type FallbackStoryGenerator struct{}

// GenerateFallbackStory creates a simple story when AI fails
func (f *FallbackStoryGenerator) GenerateFallbackStory(genre, author string) (AIResponse, error) {
	rand.Seed(time.Now().UnixNano())

	storyTemplates := map[string][]string{
		"fantasy": {
			"You find yourself in a mystical forest where ancient trees whisper secrets of old. A glowing sword calls to you from a nearby pedestal. As you approach, you hear the voice of %s echoing through the woods: 'Choose wisely, adventurer, for this blade holds great power.'",
			"In the kingdom of Eldoria, you discover an ancient prophecy hidden in the royal library. The words speak of a hero who will rise when darkness threatens the land. As you read the final passage, thunder rumbles outside. %s appears before you, eyes gleaming with ancient wisdom.",
			"The dragon's lair is filled with glittering treasures, but as you reach for a golden chalice, you hear a voice: 'Foolish mortal,' says %s, the ancient guardian. 'You have trespassed in my domain. What bargain do you offer for your life?'",
		},
		"sci-fi": {
			"The starship's AI core begins to glow with an otherworldly light. 'Warning: Unknown energy signature detected,' announces the computer. Suddenly, %s materializes in the control room, their form shifting between solid and energy states.",
			"On the distant planet of Zorath-7, you discover an ancient alien artifact buried beneath the red sands. As you brush away the dirt, holographic images appear showing %s, a legendary explorer from a long-extinct civilization.",
			"The hyperspace jump goes wrong, and you find yourself in a pocket dimension. Floating before you is %s, an entity composed entirely of quantum fluctuations. 'You have entered the space between spaces,' it intones.",
		},
		"historical-fiction": {
			"The year is 1347, and the Black Death has just arrived in your village. As you tend to the sick, you meet %s, a mysterious healer with knowledge that seems to come from beyond your time. 'I can teach you ways to fight this plague,' they whisper.",
			"In the court of King Arthur, you discover a hidden chamber beneath the Round Table. There, illuminated by torchlight, stands %s, a figure from the mists of legend. 'The old ways are fading,' they say. 'Will you help preserve them?'",
			"During the Renaissance in Florence, you stumble upon a secret society meeting in the catacombs beneath the city. At the center stands %s, their eyes reflecting the wisdom of ages past. 'Art and science shall change the world,' they proclaim.",
		},
	}

	templates, exists := storyTemplates[genre]
	if !exists {
		templates = storyTemplates["fantasy"] // Default to fantasy
	}

	selectedTemplate := templates[rand.Intn(len(templates))]
	storyText := fmt.Sprintf(selectedTemplate, author)

	response := AIResponse{
		NewGameState: &story.GameState{
			PlayerStatus: story.PlayerStatus{
				Health:     100,
				Stamina:    100,
				Conditions: []string{},
			},
			Inventory: []story.Item{
				{
					Name:        "mysterious amulet",
					Description: "a glowing pendant that hums with inner light",
					Properties:  []string{"magical"},
					State:       "equipped",
				},
			},
			Environment: story.Environment{
				LocationName: "Mysterious Chamber",
				Description:  "A dimly lit chamber filled with ancient artifacts and glowing runes.",
				Exits:        map[string]string{"north": "Dark Corridor", "east": "Treasure Room"},
				WorldObjects: []story.WorldObject{
					{Name: "ancient pedestal", Properties: []string{"stone", "carved"}, State: "stable"},
					{Name: "glowing runes", Properties: []string{"magical", "ancient"}, State: "glowing"},
				},
			},
			NPCs: []story.NPC{
				{
					Name:        author,
					Disposition: "neutral",
					Knowledge:   []string{"knows_about_ancient_secrets"},
					Goal:        "Guide the adventurer",
				},
			},
			Puzzles: []story.Puzzle{
				{
					Name:          "Ancient Riddle",
					Type:          "logic",
					Description:   "Decipher the meaning of the glowing runes",
					Status:        "unsolved",
					SolutionHints: []string{"look_closely", "think_about_symbols"},
				},
			},
			ProperNouns: []story.ProperNoun{
				{
					Noun:        author,
					PhraseUsed:  author,
					Description: "a mysterious figure with ancient knowledge",
				},
			},
			World: story.World{
				WorldTension: 25,
			},
			Rules: story.Rules{
				ConsequenceModel: "challenging",
			},
			Climax: false,
			WinConditions: []string{
				"Solve the ancient riddle",
				"Escape the chamber safely",
			},
			LossConditions: []string{
				"Trigger the chamber's traps",
				"Offend the guardian spirit",
			},
			GameWon:  false,
			GameLost: false,
		},
		StoryUpdate: StoryUpdate{
			Story:           storyText,
			ItemsAdded:      []string{"mysterious amulet"},
			ItemsRemoved:    []string{},
			GameOver:        false,
			BackgroundColor: "#2d1b69",
		},
	}

	return response, nil
}

// GetFallbackErrorMessage returns a user-friendly message when fallback is used
func GetFallbackErrorMessage() string {
	messages := []string{
		"🤖 The main story generator is currently unavailable, but here's a backup tale!",
		"📖 The AI storyteller is taking a break, but I've prepared a simple adventure!",
		"🌟 The advanced generator is offline, but enjoy this classic tale!",
		"⚔️ The primary storyteller is resting, but here's a simple quest!",
		"🏰 The main generator is updating, but here's a basic adventure!",
	}

	return messages[rand.Intn(len(messages))]
}
