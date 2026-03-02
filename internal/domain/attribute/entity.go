package attribute

type Attribute struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	IsVariantDimension bool   `json:"is_variant_dimension"`
}

type AttributeValue struct {
	ID          int64  `json:"id"`
	AttributeID int64  `json:"attribute_id"`
	Value       string `json:"value"`
}

type AttributeWithValues struct {
	Attribute
	Values []AttributeValue `json:"values"`
}