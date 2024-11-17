package models

type VectorTileStyle struct {
	Version int                     `json:"version"`
	Sources map[string]VectorSource `json:"sources"`
	Sprite  string                  `json:"sprite"`
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
	Layout      SymbolLayerLayout `json:"layout"`
	Paint       SymbolLayerPaint  `json:"paint"`
}

type SymbolLayerLayout struct {
	IconImage           string  `json:"icon-image"`
	IconSize            float64 `json:"icon-size"`
	IconAllowOverlap    bool    `json:"icon-allow-overlap"`
	IconIgnorePlacement bool    `json:"icon-ignore-placement"`
	IconOffset          []int   `json:"icon-offset"`
}

type SymbolLayerPaint struct {
	IconColor string `json:"icon-color"`
}
