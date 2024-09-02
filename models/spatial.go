package models

type GeometryInput struct {
	Geometry       string `json:"geometry" validate:"required"`
	ReturnGeometry bool   `json:"returnGeometry"`
	OutFields      string `json:"outFields"`
}
