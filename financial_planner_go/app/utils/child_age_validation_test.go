package utils

import (
	"strings"
	"testing"
)

func TestChildAgeAllowsMinimumEighteenYearCustomerGap(t *testing.T) {
	children := []ChildDetail{
		{
			Index: 1,
			Name:  "Aarav",
			Age:   22,
		},
	}

	customerAge := 40

	err := validateChildren(
		children,
		1,
		customerAge,
		"single",
		0,
	)

	if err != nil {
		t.Fatalf(
			"expected an exact 18-year customer-child age gap to pass, got: %v",
			err,
		)
	}
}

func TestChildAgeRejectsCustomerGapBelowEighteenYears(t *testing.T) {
	children := []ChildDetail{
		{
			Index: 1,
			Name:  "Aarav",
			Age:   23,
		},
	}

	customerAge := 40

	err := validateChildren(
		children,
		1,
		customerAge,
		"single",
		0,
	)

	if err == nil {
		t.Fatal(
			"expected customer-child age-gap validation error, got nil",
		)
	}

	expectedMessage :=
		"customer age must be at least 18 years greater than child age"

	if !strings.Contains(err.Error(), expectedMessage) {
		t.Fatalf(
			"expected error containing %q, got: %v",
			expectedMessage,
			err,
		)
	}
}

func TestChildAgeAllowsMinimumEighteenYearSpouseGap(t *testing.T) {
	children := []ChildDetail{
		{
			Index: 1,
			Name:  "Aarav",
			Age:   18,
		},
	}

	customerAge := 40
	spouseAge := 36

	err := validateChildren(
		children,
		1,
		customerAge,
		"married",
		spouseAge,
	)

	if err != nil {
		t.Fatalf(
			"expected customer and spouse age gaps to pass, got: %v",
			err,
		)
	}
}

func TestChildAgeRejectsSpouseGapBelowEighteenYears(t *testing.T) {
	children := []ChildDetail{
		{
			Index: 1,
			Name:  "Aarav",
			Age:   18,
		},
	}

	customerAge := 45
	spouseAge := 34

	err := validateChildren(
		children,
		1,
		customerAge,
		"married",
		spouseAge,
	)

	if err == nil {
		t.Fatal(
			"expected spouse-child age-gap validation error, got nil",
		)
	}

	expectedMessage :=
		"spouse age must be at least 18 years greater than child age"

	if !strings.Contains(err.Error(), expectedMessage) {
		t.Fatalf(
			"expected error containing %q, got: %v",
			expectedMessage,
			err,
		)
	}
}

func TestSingleCustomerSkipsSpouseChildAgeGap(t *testing.T) {
	children := []ChildDetail{
		{
			Index: 1,
			Name:  "Aarav",
			Age:   18,
		},
	}

	customerAge := 40

	err := validateChildren(
		children,
		1,
		customerAge,
		"single",
		0,
	)

	if err != nil {
		t.Fatalf(
			"expected spouse validation to be skipped for single customer, got: %v",
			err,
		)
	}
}
