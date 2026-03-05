package handlers

import "github.com/google/generative-ai-go/genai"

// BuildAIResponseSchema constructs the JSON schema expected from the model.
func BuildAIResponseSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"new_game_state": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"player_status": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"health":     {Type: genai.TypeInteger},
							"stamina":    {Type: genai.TypeInteger},
							"conditions": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
						},
					},
					"inventory": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"name":        {Type: genai.TypeString},
								"description": {Type: genai.TypeString},
								"properties":  {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
								"state":       {Type: genai.TypeString},
							},
						},
					},
					"environment": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"location_name": {Type: genai.TypeString},
							"description":   {Type: genai.TypeString},
							"exits": {
								Type: genai.TypeObject,
								Properties: map[string]*genai.Schema{
									"north":     {Type: genai.TypeString},
									"south":     {Type: genai.TypeString},
									"east":      {Type: genai.TypeString},
									"west":      {Type: genai.TypeString},
									"up":        {Type: genai.TypeString},
									"down":      {Type: genai.TypeString},
									"in":        {Type: genai.TypeString},
									"out":       {Type: genai.TypeString},
									"northeast": {Type: genai.TypeString},
									"northwest": {Type: genai.TypeString},
									"southeast": {Type: genai.TypeString},
									"southwest": {Type: genai.TypeString},
								},
							},
							"world_objects": {
								Type: genai.TypeArray,
								Items: &genai.Schema{
									Type: genai.TypeObject,
									Properties: map[string]*genai.Schema{
										"name":       {Type: genai.TypeString},
										"properties": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
										"state":      {Type: genai.TypeString},
									},
								},
							},
						},
					},
					"world": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"world_tension": {Type: genai.TypeInteger},
						},
					},
					"npcs": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"name":        {Type: genai.TypeString},
								"disposition": {Type: genai.TypeString},
								"knowledge":   {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
								"goal":        {Type: genai.TypeString},
							},
						},
					},
					"active_puzzles_and_obstacles": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"name":           {Type: genai.TypeString},
								"type":           {Type: genai.TypeString},
								"description":    {Type: genai.TypeString},
								"status":         {Type: genai.TypeString},
								"solution_hints": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
							},
						},
					},
					"proper_nouns": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"noun":        {Type: genai.TypeString},
								"phrase_used": {Type: genai.TypeString},
								"description": {Type: genai.TypeString},
							},
						},
					},
					"rules": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"consequence_model": {Type: genai.TypeString},
						},
					},
					"climax":              {Type: genai.TypeBoolean},
					"win_conditions":      {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
					"loss_conditions":     {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
					"game_won":            {Type: genai.TypeBoolean},
					"game_lost":           {Type: genai.TypeBoolean},
					"solved_puzzle_types": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
				},
			},
			"story_update": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"story":            {Type: genai.TypeString},
					"items_added":      {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
					"items_removed":    {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
					"game_over":        {Type: genai.TypeBoolean},
					"background_color": {Type: genai.TypeString},
				},
			},
		},
	}
}
