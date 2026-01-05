package models

type VectorTileStyle struct {
	Version int                     `json:"version"`
	Sources map[string]VectorSource `json:"sources"`
	Sprite  string                  `json:"sprite"`
	Glyphs  string                  `json:"glyphs"`
	Layers  []any                   `json:"layers"`
}

type VectorSource struct {
	Type  string   `json:"type"`
	Tiles []string `json:"tiles"`
}

// Fill layer struct
type FillLayer struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Source      string         `json:"source"`
	SourceLayer string         `json:"source-layer"`
	Paint       FillLayerPaint `json:"paint"`
}

type FillLayerPaint struct {
	FillColor   string  `json:"fill-color"`
	FillOpacity float64 `json:"fill-opacity"`
}

// Line layer struct
type LineLayer struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Source      string         `json:"source"`
	SourceLayer string         `json:"source-layer"`
	Paint       LineLayerPaint `json:"paint"`
}

type LineLayerPaint struct {
	LineColor string  `json:"line-color"`
	LineWidth float64 `json:"line-width"`
}

// Symbol layer struct
type SymbolLayer struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Source      string            `json:"source"`
	SourceLayer string            `json:"source-layer"`
	Filter      []interface{}     `json:"filter,omitempty"`
	Layout      SymbolLayerLayout `json:"layout"`
	Paint       SymbolLayerPaint  `json:"paint"`
}

type SymbolLayerLayout struct {
	IconImage           string        `json:"icon-image,omitempty"`
	IconSize            float64       `json:"icon-size,omitempty"`
	IconAllowOverlap    bool          `json:"icon-allow-overlap,omitempty"`
	IconIgnorePlacement bool          `json:"icon-ignore-placement,omitempty"`
	IconOffset          []int         `json:"icon-offset,omitempty"`
	TextField           []interface{} `json:"text-field,omitempty"`
	TextFont            []string      `json:"text-font,omitempty"`
	TextSize            float64       `json:"text-size,omitempty"`
	TextOffset          []float64     `json:"text-offset,omitempty"`
	TextAnchor          string        `json:"text-anchor,omitempty"`
}

type SymbolLayerPaint struct {
	IconColor string `json:"icon-color,omitempty"`
	TextColor string `json:"text-color,omitempty"`
}

// Circle layer struct for clustering
type CircleLayer struct {
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	Source      string           `json:"source"`
	SourceLayer string           `json:"source-layer"`
	Filter      []interface{}    `json:"filter,omitempty"`
	Paint       CircleLayerPaint `json:"paint"`
}

type CircleLayerPaint struct {
	CircleColor         interface{} `json:"circle-color"`
	CircleRadius        interface{} `json:"circle-radius"`
	CircleStrokeOpacity float64     `json:"circle-stroke-opacity,omitempty"`
	CircleOpacity       float64     `json:"circle-opacity,omitempty"`
	CircleStrokeWidth   float64     `json:"circle-stroke-width,omitempty"`
	CircleStrokeColor   string      `json:"circle-stroke-color,omitempty"`
}
