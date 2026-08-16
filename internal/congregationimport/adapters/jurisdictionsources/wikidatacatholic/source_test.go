// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package wikidatacatholic

import "testing"

func TestQidFromURI(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"http://www.wikidata.org/entity/Q40741", "Q40741"},
		{"http://www.wikidata.org/entity/Q1", "Q1"},
		{"", ""},
		{"http://www.wikidata.org/entity/statement/Q40741-abc", ""}, // not a bare entity URI
	}
	for _, tt := range tests {
		if got := qidFromURI(tt.uri); got != tt.want {
			t.Errorf("qidFromURI(%q) = %q, want %q", tt.uri, got, tt.want)
		}
	}
}

func TestPrimaryAndAliases(t *testing.T) {
	name, aliases := primaryAndAliases([]string{"Roman Catholic Archdiocese of Lviv", "Львівська архієпархія"})
	if name != "Roman Catholic Archdiocese of Lviv" {
		t.Errorf("primary = %q, want %q", name, "Roman Catholic Archdiocese of Lviv")
	}
	if len(aliases) != 1 || aliases[0] != "Львівська архієпархія" {
		t.Errorf("aliases = %v, want [Львівська архієпархія]", aliases)
	}
}

func TestPrimaryAndAliasesEmpty(t *testing.T) {
	name, aliases := primaryAndAliases(nil)
	if name != "" || aliases != nil {
		t.Errorf("primaryAndAliases(nil) = (%q, %v), want (\"\", nil)", name, aliases)
	}
}

func TestNewRejectsMalformedCountryQID(t *testing.T) {
	if _, err := New("", []string{"not-a-qid"}, nil); err == nil {
		t.Error("New must reject a malformed country QID rather than silently interpolating it into a SPARQL query")
	}
}

func TestNewAcceptsValidCountryQIDs(t *testing.T) {
	s, err := New("", []string{"Q212", "Q30"}, nil)
	if err != nil {
		t.Fatalf("New(valid QIDs) returned an error: %v", err)
	}
	if s.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %q, want default %q", s.BaseURL, defaultBaseURL)
	}
}
