package importer

// FoundationFood represents a food item from USDA Foundation Foods JSON
type FoundationFood struct {
	FdcId            int             `json:"fdcId"`
	Description      string          `json:"description"`
	DataType         string          `json:"dataType"`
	FoodClass        string          `json:"foodClass"`
	PublicationDate  string          `json:"publicationDate"`
	FoodNutrients    []FoodNutrient  `json:"foodNutrients"`
	AllNutrientNames []string        `json:"allNutrientNames,omitempty"`
	InputFoods       []InputFood     `json:"inputFoods,omitempty"`
	FoodPortions     []FoodPortion   `json:"foodPortions"`
	FoodAttributes   []FoodAttribute `json:"foodAttributes"`
}

type FoodNutrient struct {
	Type                   string      `json:"type"`
	Id                     int         `json:"id"`
	Nutrient               Nutrient    `json:"nutrient"`
	Amount                 float64     `json:"amount,omitempty"`
	DataPoints             int         `json:"dataPoints,omitempty"`
	Min                    float64     `json:"min,omitempty"`
	Max                    float64     `json:"max,omitempty"`
	Median                 float64     `json:"median,omitempty"`
	FoodNutrientDerivation *Derivation `json:"foodNutrientDerivation,omitempty"`
}

type Nutrient struct {
	Id       int    `json:"id"`
	Number   string `json:"number"`
	Name     string `json:"name"`
	Rank     int    `json:"rank"`
	UnitName string `json:"unitName"`
}

type Derivation struct {
	Code               string  `json:"code"`
	Description        string  `json:"description"`
	FoodNutrientSource *Source `json:"foodNutrientSource"`
}

type Source struct {
	Id          int    `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

type FoodPortion struct {
	Id           int     `json:"id"`
	SeqNum       int     `json:"seqNum"`
	Amount       float64 `json:"amount"`
	UnitName     string  `json:"unitName"`
	Grams        float64 `json:"gramWeight"`
	DataPoints   int     `json:"dataPoints"`
	DerivationId string  `json:"derivationId"`
	PortionName  string  `json:"portionName"`
	PortionDesc  string  `json:"portionDescription"`
}

type FoodAttribute struct {
	SeqNum       int    `json:"seqNum"`
	Name         string `json:"name"`
	Value        string `json:"value"`
	Unit         string `json:"unit"`
	DataType     string `json:"dataType"`
	DerivationId string `json:"derivationId"`
}

type InputFood struct {
	SrcName  string `json:"srcName"`
	SrcId    int    `json:"srcId"`
	SrcTable string `json:"srcTable"`
	SrcDate  string `json:"srcDate"`
}