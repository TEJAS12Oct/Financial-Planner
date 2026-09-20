package utils

import (
	"sort"
	"strings"
)

type GoalDefinition struct {
	Key       string
	Label     string
	Type      string
	MinMonths int
	MaxMonths int
	MinYears  int
	MaxYears  int
}

var goalDefinitions = map[string]GoalDefinition{
	"iphone": {
		Key:       "iphone",
		Label:     "Phone",
		Type:      "instant",
		MinMonths: 6,
		MaxMonths: 12,
	},
	"playstation": {
		Key:       "playstation",
		Label:     "PlayStation",
		Type:      "instant",
		MinMonths: 6,
		MaxMonths: 12,
	},
	"bike": {
		Key:       "bike",
		Label:     "Bike",
		Type:      "instant",
		MinMonths: 6,
		MaxMonths: 12,
	},
	"tv": {
		Key:       "tv",
		Label:     "TV",
		Type:      "instant",
		MinMonths: 6,
		MaxMonths: 12,
	},
	"fridge": {
		Key:       "fridge",
		Label:     "Fridge",
		Type:      "instant",
		MinMonths: 6,
		MaxMonths: 12,
	},
	"washing_machine": {
		Key:       "washing_machine",
		Label:     "Washing Machine",
		Type:      "instant",
		MinMonths: 6,
		MaxMonths: 12,
	},
	"vacation": {
		Key:       "vacation",
		Label:     "Vacation Plan",
		Type:      "instant",
		MinMonths: 6,
		MaxMonths: 12,
	},
	"others": {
		Key:       "others",
		Label:     "Others",
		Type:      "instant",
		MinMonths: 6,
		MaxMonths: 12,
	},
	"higher_education": {
		Key:      "higher_education",
		Label:    "Higher Education",
		Type:     "short",
		MinYears: 1,
		MaxYears: 5,
	},
	"home_purchase": {
		Key:      "home_purchase",
		Label:    "Home Purchase / Home Upgrade",
		Type:     "short",
		MinYears: 1,
		MaxYears: 5,
	},
	"car_purchase": {
		Key:      "car_purchase",
		Label:    "Car Purchase",
		Type:     "short",
		MinYears: 1,
		MaxYears: 5,
	},
	"own_marriage": {
		Key:      "own_marriage",
		Label:    "Own Marriage Planning",
		Type:     "short",
		MinYears: 1,
		MaxYears: 5,
	},
	"retirement": {
		Key:      "retirement",
		Label:    "Retirement Planning",
		Type:     "long",
		MinYears: 20,
		MaxYears: 25,
	},
	"wealth": {
		Key:      "wealth",
		Label:    "Wealth Creation",
		Type:     "long",
		MinYears: 20,
		MaxYears: 25,
	},
}

func GetGoalDefinition(
	key string,
) (GoalDefinition, bool) {
	normalizedKey := strings.ToLower(
		strings.TrimSpace(key),
	)

	definition, exists :=
		goalDefinitions[normalizedKey]

	return definition, exists
}

func IsValidGoalKey(
	key string,
) bool {
	_, exists :=
		GetGoalDefinition(key)

	return exists
}

func IsValidGoalLabel(
	key string,
	label string,
) bool {
	definition, exists :=
		GetGoalDefinition(key)

	if !exists {
		return false
	}

	return strings.EqualFold(
		strings.TrimSpace(label),
		definition.Label,
	)
}

func IsValidGoalType(
	key string,
	goalType string,
) bool {
	definition, exists :=
		GetGoalDefinition(key)

	if !exists {
		return false
	}

	return strings.EqualFold(
		strings.TrimSpace(goalType),
		definition.Type,
	)
}

func SupportedGoalKeys() []string {
	keys := make(
		[]string,
		0,
		len(goalDefinitions),
	)

	for key := range goalDefinitions {
		keys = append(
			keys,
			key,
		)
	}

	sort.Strings(keys)

	return keys
}

func SupportedGoalKeysByType(
	goalType string,
) []string {
	normalizedType := strings.ToLower(
		strings.TrimSpace(goalType),
	)

	keys := make(
		[]string,
		0,
	)

	for key, definition := range goalDefinitions {

		if definition.Type ==
			normalizedType {

			keys = append(
				keys,
				key,
			)
		}
	}

	sort.Strings(keys)

	return keys
}

func SupportedGoalLabelsByType(
	goalType string,
) []string {
	normalizedType := strings.ToLower(
		strings.TrimSpace(goalType),
	)

	labels := make(
		[]string,
		0,
	)

	for _, definition := range goalDefinitions {

		if definition.Type ==
			normalizedType {

			labels = append(
				labels,
				definition.Label,
			)
		}
	}

	sort.Strings(labels)

	return labels
}

func SupportedGoalTypes() []string {
	return []string{
		"instant",
		"short",
		"long",
	}
}
