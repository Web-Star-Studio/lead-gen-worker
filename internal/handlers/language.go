package handlers

import (
	"strings"
	"unicode"

	"webstar/noturno-leadgen-worker/internal/dto"
)

// Language constants
const (
	LangPortuguese = "pt-BR"
	LangEnglish    = "en"
)

// brazilianLocations contains common Brazilian location indicators
var brazilianLocations = []string{
	"brazil", "brasil",
	"são paulo", "sao paulo", "sp",
	"rio de janeiro", "rj",
	"belo horizonte", "mg", "minas gerais",
	"salvador", "ba", "bahia",
	"brasília", "brasilia", "df",
	"curitiba", "pr", "paraná", "parana",
	"recife", "pe", "pernambuco",
	"fortaleza", "ce", "ceará", "ceara",
	"porto alegre", "rs", "rio grande do sul",
	"manaus", "am", "amazonas",
	"belém", "belem", "pa", "pará", "para",
	"goiânia", "goiania", "go", "goiás", "goias",
	"campinas", "santos", "guarulhos",
	"florianópolis", "florianopolis", "sc", "santa catarina",
	"vitória", "vitoria", "es", "espírito santo", "espirito santo",
	"natal", "rn", "rio grande do norte",
	"joão pessoa", "joao pessoa", "pb", "paraíba", "paraiba",
	"maceió", "maceio", "al", "alagoas",
	"teresina", "pi", "piauí", "piaui",
	"campo grande", "ms", "mato grosso do sul",
	"cuiabá", "cuiaba", "mt", "mato grosso",
	"aracaju", "se", "sergipe",
	"são luís", "sao luis", "ma", "maranhão", "maranhao",
	"porto velho", "ro", "rondônia", "rondonia",
	"macapá", "macapa", "ap", "amapá", "amapa",
	"boa vista", "rr", "roraima",
	"palmas", "to", "tocantins",
	"rio branco", "ac", "acre",
}

// DetectLanguage determines the output language based on business profile and location
// Returns "en" for English or "pt-BR" for Portuguese
func DetectLanguage(profile *dto.BusinessProfile, location string) string {
	// 1. If profile has explicit language set, use it
	if profile != nil && profile.Language != "" {
		lang := strings.ToLower(profile.Language)
		if strings.Contains(lang, "en") {
			return LangEnglish
		}
		if strings.Contains(lang, "pt") {
			return LangPortuguese
		}
	}

	// 2. Check if profile content is in English (heuristic based on text analysis)
	if profile != nil && isEnglishContent(profile) {
		return LangEnglish
	}

	// 3. Check if location is NOT in Brazil
	if location != "" && !isBrazilianLocation(location) {
		return LangEnglish
	}

	// Default to Portuguese for Brazilian leads
	return LangPortuguese
}

// isEnglishContent checks if the business profile content appears to be in English
func isEnglishContent(profile *dto.BusinessProfile) bool {
	// Combine relevant text fields
	content := strings.ToLower(profile.CompanyDescription + " " + profile.ProblemSolved + " " + profile.SuccessCase)
	
	if content == "" {
		return false
	}

	// Common English words that are unlikely in Portuguese business text
	englishIndicators := []string{
		" the ", " and ", " with ", " our ", " your ", " we ", " you ",
		" for ", " that ", " this ", " from ", " have ", " are ", " will ",
		" can ", " help ", " business ", " company ", " service ", " provide ",
		" solution ", " customer ", " client ", " team ", " work ",
	}

	// Common Portuguese words
	portugueseIndicators := []string{
		" que ", " para ", " com ", " uma ", " seu ", " sua ", " nos ", " nós ",
		" você ", " empresa ", " serviço ", " cliente ", " negócio ", " solução ",
		" nossa ", " nosso ", " trabalho ", " equipe ", " ajuda ", " oferece ",
		" através ", " sobre ", " como ", " mais ", " está ", " são ", " pelo ",
	}

	englishScore := 0
	portugueseScore := 0

	for _, word := range englishIndicators {
		if strings.Contains(content, word) {
			englishScore++
		}
	}

	for _, word := range portugueseIndicators {
		if strings.Contains(content, word) {
			portugueseScore++
		}
	}

	// Also check for Portuguese-specific characters (accents)
	for _, r := range content {
		if r == 'ã' || r == 'õ' || r == 'ç' || r == 'é' || r == 'ê' || r == 'á' || r == 'ó' || r == 'ú' || r == 'í' {
			portugueseScore++
		}
	}

	// If significantly more English indicators, consider it English
	return englishScore > portugueseScore+2
}

// isBrazilianLocation checks if the location string indicates Brazil
func isBrazilianLocation(location string) bool {
	locationLower := strings.ToLower(location)
	
	// Remove accents for comparison
	locationNormalized := removeAccents(locationLower)

	for _, loc := range brazilianLocations {
		if strings.Contains(locationLower, loc) || strings.Contains(locationNormalized, loc) {
			return true
		}
	}

	return false
}

// removeAccents removes common Portuguese accents from a string
func removeAccents(s string) string {
	var result strings.Builder
	for _, r := range s {
		switch r {
		case 'á', 'à', 'ã', 'â', 'ä':
			result.WriteRune('a')
		case 'é', 'è', 'ê', 'ë':
			result.WriteRune('e')
		case 'í', 'ì', 'î', 'ï':
			result.WriteRune('i')
		case 'ó', 'ò', 'õ', 'ô', 'ö':
			result.WriteRune('o')
		case 'ú', 'ù', 'û', 'ü':
			result.WriteRune('u')
		case 'ç':
			result.WriteRune('c')
		default:
			if unicode.IsLetter(r) || unicode.IsSpace(r) || unicode.IsDigit(r) {
				result.WriteRune(r)
			}
		}
	}
	return result.String()
}
