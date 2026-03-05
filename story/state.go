package story

// ProperNoun represents a noun and its description for tooltip generation.
type ProperNoun struct {
	Noun        string `json:"noun"`
	PhraseUsed  string `json:"phrase"`
	Description string `json:"desc"`
}

// GameState represents the entire state of the game world.
type GameState struct {
	PlayerStatus      PlayerStatus `json:"status"`
	Inventory         []Item       `json:"inv,omitempty"`
	Environment       Environment  `json:"env"`
	World             World        `json:"world"`
	NPCs              []NPC        `json:"npcs,omitempty"`
	Puzzles           []Puzzle     `json:"puzzles,omitempty"`
	ProperNouns       []ProperNoun `json:"nouns,omitempty"`
	Rules             Rules        `json:"rules"`
	Climax            bool         `json:"climax"`
	WinConditions     []string     `json:"win,omitempty"`
	LossConditions    []string     `json:"loss,omitempty"`
	GameWon           bool         `json:"won"`
	GameLost          bool         `json:"lost"`
	SolvedPuzzleTypes []string     `json:"solved_puzzles,omitempty"`
}

// PlayerStatus tracks the player's condition.
type PlayerStatus struct {
	Health     int      `json:"hp"`
	Stamina    int      `json:"sp"`
	Conditions []string `json:"conds,omitempty"`
}

// Item represents an object in the player's inventory.
type Item struct {
	Name        string   `json:"name"`
	Description string   `json:"desc"`
	Properties  []string `json:"props,omitempty"`
	State       string   `json:"state,omitempty"`
}

// Environment describes the current location and its interactive elements.
type Environment struct {
	LocationName string            `json:"loc"`
	Description  string            `json:"desc"`
	Exits        map[string]string `json:"exits,omitempty"`
	WorldObjects []WorldObject     `json:"objs,omitempty"`
}

// WorldObject represents an interactable object in the environment.
type WorldObject struct {
	Name       string   `json:"name"`
	Properties []string `json:"props,omitempty"`
	State      string   `json:"state,omitempty"`
}

// NPC represents a non-player character.
type NPC struct {
	Name        string   `json:"name"`
	Disposition string   `json:"disp"`
	Knowledge   []string `json:"know,omitempty"`
	Goal        string   `json:"goal"`
}

// Puzzle represents an active challenge for the player.
type Puzzle struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Description   string   `json:"desc"`
	Status        string   `json:"status"`
	SolutionHints []string `json:"hints,omitempty"`
}

// World represents the global state of the world.
type World struct {
	WorldTension int `json:"tension"`
}

// Rules defines the current rule set for the game.
type Rules struct {
	ConsequenceModel string `json:"model"`
}
