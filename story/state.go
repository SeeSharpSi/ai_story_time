package story

// GameState represents the entire state of the game world.
type GameState struct {
	PlayerStatus PlayerStatus `json:"player_status"`
	Inventory    []Item       `json:"inventory"`
	Environment  Environment  `json:"environment"`
	NPCs         []NPC        `json:"npcs"`
	Puzzles      []Puzzle     `json:"active_puzzles_and_obstacles"`
	Rules        Rules        `json:"rules"`
}

// PlayerStatus tracks the player's condition.
type PlayerStatus struct {
	Health     int      `json:"health"`
	Stamina    int      `json:"stamina"`
	Conditions []string `json:"conditions"`
}

// Item represents an object in the player's inventory.
type Item struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Properties  []string `json:"properties"`
	State       string   `json:"state"`
}

// Environment describes the current location and its interactive elements.
type Environment struct {
	LocationName string        `json:"location_name"`
	Description  string        `json:"description"`
	Exits        map[string]string `json:"exits"`
	WorldObjects []WorldObject `json:"world_objects"`
}

// WorldObject represents an interactable object in the environment.
type WorldObject struct {
	Name       string   `json:"name"`
	Properties []string `json:"properties"`
	State      string   `json:"state"`
}

// NPC represents a non-player character.
type NPC struct {
	Name        string   `json:"name"`
	Disposition string   `json:"disposition"`
	Knowledge   []string `json:"knowledge"`
	Goal        string   `json:"goal"`
}

// Puzzle represents an active challenge for the player.
type Puzzle struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	SolutionHints []string `json:"solution_hints"`
}

// Rules defines the current rule set for the game.
type Rules struct {
	ConsequenceModel string `json:"consequence_model"`
}
