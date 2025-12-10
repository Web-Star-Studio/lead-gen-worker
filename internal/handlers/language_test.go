package handlers

import (
	"testing"

	"webstar/noturno-leadgen-worker/internal/dto"

	"github.com/stretchr/testify/assert"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		profile  *dto.BusinessProfile
		location string
		expected string
	}{
		{
			name:     "nil profile, empty location - default Portuguese",
			profile:  nil,
			location: "",
			expected: LangPortuguese,
		},
		{
			name: "profile with explicit Portuguese",
			profile: &dto.BusinessProfile{
				Language: "pt-BR",
			},
			location: "",
			expected: LangPortuguese,
		},
		{
			name: "profile with explicit English",
			profile: &dto.BusinessProfile{
				Language: "en-US",
			},
			location: "",
			expected: LangEnglish,
		},
		{
			name: "profile with english content",
			profile: &dto.BusinessProfile{
				CompanyDescription: "We are a company that provides the best solutions for your business needs",
				ProblemSolved:      "We help companies solve their problems with our services",
			},
			location: "",
			expected: LangEnglish,
		},
		{
			name: "profile with Portuguese content",
			profile: &dto.BusinessProfile{
				CompanyDescription: "Somos uma empresa que oferece as melhores soluções para sua empresa",
				ProblemSolved:      "Nós ajudamos empresas a resolver seus problemas através dos nossos serviços",
			},
			location: "",
			expected: LangPortuguese,
		},
		{
			name:     "location in São Paulo",
			profile:  nil,
			location: "São Paulo, SP",
			expected: LangPortuguese,
		},
		{
			name:     "location in Recife",
			profile:  nil,
			location: "Recife, Pernambuco",
			expected: LangPortuguese,
		},
		{
			name:     "location in USA",
			profile:  nil,
			location: "New York, USA",
			expected: LangEnglish,
		},
		{
			name:     "location in London",
			profile:  nil,
			location: "London, UK",
			expected: LangEnglish,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectLanguage(tt.profile, tt.location)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBrazilianLocation(t *testing.T) {
	tests := []struct {
		name     string
		location string
		expected bool
	}{
		{
			name:     "São Paulo",
			location: "São Paulo, SP",
			expected: true,
		},
		{
			name:     "Sao Paulo without accent",
			location: "Sao Paulo",
			expected: true,
		},
		{
			name:     "Rio de Janeiro",
			location: "Rio de Janeiro, RJ",
			expected: true,
		},
		{
			name:     "Recife",
			location: "Recife - PE",
			expected: true,
		},
		{
			name:     "Brazil country",
			location: "Brazil",
			expected: true,
		},
		{
			name:     "Brasil",
			location: "Brasil",
			expected: true,
		},
		{
			name:     "New York",
			location: "New York, USA",
			expected: false,
		},
		{
			name:     "London",
			location: "London, UK",
			expected: false,
		},
		{
			name:     "empty location",
			location: "",
			expected: false,
		},
		{
			name:     "Belo Horizonte",
			location: "Belo Horizonte, MG",
			expected: true,
		},
		{
			name:     "Florianópolis",
			location: "Florianópolis, SC",
			expected: true,
		},
		{
			name:     "Fortaleza",
			location: "Fortaleza, Ceará",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBrazilianLocation(tt.location)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsEnglishContent(t *testing.T) {
	tests := []struct {
		name     string
		profile  *dto.BusinessProfile
		expected bool
	}{
		{
			name: "English content",
			profile: &dto.BusinessProfile{
				CompanyDescription: "We provide the best solutions for your business. Our team will help you achieve your goals.",
				ProblemSolved:      "We help companies that are struggling with their operations.",
			},
			expected: true,
		},
		{
			name: "Portuguese content",
			profile: &dto.BusinessProfile{
				CompanyDescription: "Nós oferecemos as melhores soluções para sua empresa. Nossa equipe vai ajudar você.",
				ProblemSolved:      "Ajudamos empresas que estão com dificuldades nas suas operações.",
			},
			expected: false,
		},
		{
			name: "Portuguese with accents",
			profile: &dto.BusinessProfile{
				CompanyDescription: "Somos especialistas em soluções tecnológicas para o mercado brasileiro.",
				ProblemSolved:      "Resolvemos problemas através da inovação e qualidade.",
			},
			expected: false,
		},
		{
			name: "empty content",
			profile: &dto.BusinessProfile{
				CompanyDescription: "",
				ProblemSolved:      "",
			},
			expected: false,
		},
		{
			name: "mixed content - more English",
			profile: &dto.BusinessProfile{
				CompanyDescription: "We are the best company for your needs. Our solution will help you grow and achieve success.",
				ProblemSolved:      "Companies that need help with their business operations.",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEnglishContent(tt.profile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveAccents(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "São Paulo",
			input:    "são paulo",
			expected: "sao paulo",
		},
		{
			name:     "Florianópolis",
			input:    "florianópolis",
			expected: "florianopolis",
		},
		{
			name:     "múltiple áccénts",
			input:    "múltiple áccénts",
			expected: "multiple accents",
		},
		{
			name:     "ç cedilha",
			input:    "ação",
			expected: "acao",
		},
		{
			name:     "all Portuguese accents",
			input:    "áàãâä éèêë íìîï óòõôö úùûü ç",
			expected: "aaaaa eeee iiii ooooo uuuu c",
		},
		{
			name:     "no accents",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "with numbers",
			input:    "são paulo 2024",
			expected: "sao paulo 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeAccents(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLanguageConstants(t *testing.T) {
	assert.Equal(t, "pt-BR", LangPortuguese)
	assert.Equal(t, "en", LangEnglish)
}

func TestBrazilianLocationsComprehensive(t *testing.T) {
	// Test all state abbreviations
	stateAbbreviations := []string{
		"sp", "rj", "mg", "ba", "df", "pr", "pe", "ce", "rs", "am",
		"pa", "go", "sc", "es", "rn", "pb", "al", "pi", "ms", "mt",
		"se", "ma", "ro", "ap", "rr", "to", "ac",
	}

	for _, abbr := range stateAbbreviations {
		t.Run("state_"+abbr, func(t *testing.T) {
			assert.True(t, isBrazilianLocation(abbr), "Should recognize %s as Brazilian location", abbr)
		})
	}
}
