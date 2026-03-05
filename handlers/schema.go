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
					"status": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"hp":    {Type: genai.TypeInteger},
							"sp":    {Type: genai.TypeInteger},
							"conds": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
						},
					},
					"inv": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"name":  {Type: genai.TypeString},
								"desc":  {Type: genai.TypeString},
								"props": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
								"state": {Type: genai.TypeString},
							},
						},
					},
					"env": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"loc":  {Type: genai.TypeString},
							"desc": {Type: genai.TypeString},
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
							"objs": {
								Type: genai.TypeArray,
								Items: &genai.Schema{
									Type: genai.TypeObject,
									Properties: map[string]*genai.Schema{
										"name":  {Type: genai.TypeString},
										"props": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
										"state": {Type: genai.TypeString},
									},
								},
							},
						},
					},
					"world": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"tension": {Type: genai.TypeInteger},
						},
					},
					"npcs": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"name": {Type: genai.TypeString},
								"disp": {Type: genai.TypeString},
								"know": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
								"goal": {Type: genai.TypeString},
							},
						},
					},
					"puzzles": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"name":   {Type: genai.TypeString},
								"type":   {Type: genai.TypeString},
								"desc":   {Type: genai.TypeString},
								"status": {Type: genai.TypeString},
								"hints":  {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
							},
						},
					},
					"nouns": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"noun":   {Type: genai.TypeString},
								"phrase": {Type: genai.TypeString},
								"desc":   {Type: genai.TypeString},
							},
						},
					},
					"rules": {
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"model": {Type: genai.TypeString},
						},
					},
					"climax":         {Type: genai.TypeBoolean},
					"win":            {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
					"loss":           {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
					"won":            {Type: genai.TypeBoolean},
					"lost":           {Type: genai.TypeBoolean},
					"solved_puzzles": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
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
