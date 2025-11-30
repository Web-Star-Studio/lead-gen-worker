package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchRequest_JSONMarshaling tests JSON marshaling of SearchRequest
func TestSearchRequest_JSONMarshaling(t *testing.T) {
	request := SearchRequest{
		Q:              "escritório de contabilidade",
		Location:       "Recife",
		Hl:             "pt-br",
		Gl:             "br",
		ExcludeDomains: []string{"instagram.com", "linkedin.com"},
		Num:            20,
		Start:          0,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(request)
	require.NoError(t, err)

	// Verify JSON contains expected fields
	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"q":"escritório de contabilidade"`)
	assert.Contains(t, jsonStr, `"location":"Recife"`)
	assert.Contains(t, jsonStr, `"hl":"pt-br"`)
	assert.Contains(t, jsonStr, `"gl":"br"`)
	assert.Contains(t, jsonStr, `"exclude_domains"`)
	assert.Contains(t, jsonStr, `"num":20`)
	assert.Contains(t, jsonStr, `"start":0`)
}

// TestSearchRequest_JSONUnmarshaling tests JSON unmarshaling of SearchRequest
func TestSearchRequest_JSONUnmarshaling(t *testing.T) {
	jsonStr := `{
		"q": "advogado trabalhista",
		"location": "São Paulo",
		"hl": "pt-br",
		"gl": "br",
		"exclude_domains": ["instagram.com", "facebook.com"],
		"num": 50,
		"start": 10
	}`

	var request SearchRequest
	err := json.Unmarshal([]byte(jsonStr), &request)
	require.NoError(t, err)

	assert.Equal(t, "advogado trabalhista", request.Q)
	assert.Equal(t, "São Paulo", request.Location)
	assert.Equal(t, "pt-br", request.Hl)
	assert.Equal(t, "br", request.Gl)
	assert.Len(t, request.ExcludeDomains, 2)
	assert.Contains(t, request.ExcludeDomains, "instagram.com")
	assert.Contains(t, request.ExcludeDomains, "facebook.com")
	assert.Equal(t, 50, request.Num)
	assert.Equal(t, 10, request.Start)
}

// TestSearchRequest_JSONUnmarshaling_RequiredFieldsOnly tests unmarshaling with only required fields
func TestSearchRequest_JSONUnmarshaling_RequiredFieldsOnly(t *testing.T) {
	jsonStr := `{
		"q": "test query",
		"location": "Recife"
	}`

	var request SearchRequest
	err := json.Unmarshal([]byte(jsonStr), &request)
	require.NoError(t, err)

	assert.Equal(t, "test query", request.Q)
	assert.Equal(t, "Recife", request.Location)
	assert.Empty(t, request.Hl)
	assert.Empty(t, request.Gl)
	assert.Nil(t, request.ExcludeDomains)
	assert.Equal(t, 0, request.Num)
	assert.Equal(t, 0, request.Start)
}

// TestSearchRequest_JSONUnmarshaling_EmptyExcludeDomains tests empty exclude_domains array
func TestSearchRequest_JSONUnmarshaling_EmptyExcludeDomains(t *testing.T) {
	jsonStr := `{
		"q": "test query",
		"location": "Recife",
		"exclude_domains": []
	}`

	var request SearchRequest
	err := json.Unmarshal([]byte(jsonStr), &request)
	require.NoError(t, err)

	assert.NotNil(t, request.ExcludeDomains)
	assert.Empty(t, request.ExcludeDomains)
}

// TestSearchRequest_DefaultValues tests default values of SearchRequest
func TestSearchRequest_DefaultValues(t *testing.T) {
	request := SearchRequest{}

	assert.Empty(t, request.Q)
	assert.Empty(t, request.Location)
	assert.Empty(t, request.Hl)
	assert.Empty(t, request.Gl)
	assert.Nil(t, request.ExcludeDomains)
	assert.Equal(t, 0, request.Num)
	assert.Equal(t, 0, request.Start)
}

// TestSearchRequest_SpecialCharacters tests handling of special characters
func TestSearchRequest_SpecialCharacters(t *testing.T) {
	request := SearchRequest{
		Q:        "café & restaurante",
		Location: "São Paulo, SP",
	}

	jsonBytes, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded SearchRequest
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "café & restaurante", decoded.Q)
	assert.Equal(t, "São Paulo, SP", decoded.Location)
}

// TestSearchRequest_UnicodeCharacters tests handling of unicode characters
func TestSearchRequest_UnicodeCharacters(t *testing.T) {
	request := SearchRequest{
		Q:        "日本語テスト",
		Location: "東京",
	}

	jsonBytes, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded SearchRequest
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "日本語テスト", decoded.Q)
	assert.Equal(t, "東京", decoded.Location)
}

// TestErrorResponse_JSONMarshaling tests JSON marshaling of ErrorResponse
func TestErrorResponse_JSONMarshaling(t *testing.T) {
	response := ErrorResponse{
		Error: "Key: 'SearchRequest.Q' Error:Field validation for 'Q' failed on the 'required' tag",
	}

	jsonBytes, err := json.Marshal(response)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"error"`)
	assert.Contains(t, jsonStr, "SearchRequest.Q")
}

// TestErrorResponse_JSONUnmarshaling tests JSON unmarshaling of ErrorResponse
func TestErrorResponse_JSONUnmarshaling(t *testing.T) {
	jsonStr := `{"error": "Internal server error"}`

	var response ErrorResponse
	err := json.Unmarshal([]byte(jsonStr), &response)
	require.NoError(t, err)

	assert.Equal(t, "Internal server error", response.Error)
}

// TestErrorResponse_EmptyError tests ErrorResponse with empty error message
func TestErrorResponse_EmptyError(t *testing.T) {
	response := ErrorResponse{Error: ""}

	jsonBytes, err := json.Marshal(response)
	require.NoError(t, err)

	assert.Contains(t, string(jsonBytes), `"error":""`)
}

// TestSearchRequest_ManyExcludeDomains tests handling of many excluded domains
func TestSearchRequest_ManyExcludeDomains(t *testing.T) {
	domains := []string{
		"instagram.com",
		"linkedin.com",
		"facebook.com",
		"twitter.com",
		"tiktok.com",
		"youtube.com",
		"pinterest.com",
		"reddit.com",
		"tumblr.com",
		"snapchat.com",
	}

	request := SearchRequest{
		Q:              "test query",
		Location:       "Recife",
		ExcludeDomains: domains,
	}

	jsonBytes, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded SearchRequest
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.ExcludeDomains, 10)
	for _, domain := range domains {
		assert.Contains(t, decoded.ExcludeDomains, domain)
	}
}

// TestSearchRequest_LargeNum tests handling of large num values
func TestSearchRequest_LargeNum(t *testing.T) {
	request := SearchRequest{
		Q:        "test query",
		Location: "Recife",
		Num:      100,
		Start:    1000,
	}

	jsonBytes, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded SearchRequest
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 100, decoded.Num)
	assert.Equal(t, 1000, decoded.Start)
}

// TestSearchRequest_ZeroValues tests handling of zero values
func TestSearchRequest_ZeroValues(t *testing.T) {
	request := SearchRequest{
		Q:        "test query",
		Location: "Recife",
		Num:      0,
		Start:    0,
	}

	jsonBytes, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded SearchRequest
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 0, decoded.Num)
	assert.Equal(t, 0, decoded.Start)
}

// TestSearchRequest_JSONFieldNames tests that JSON field names are correct
func TestSearchRequest_JSONFieldNames(t *testing.T) {
	request := SearchRequest{
		Q:              "test",
		Location:       "loc",
		Hl:             "en",
		Gl:             "us",
		ExcludeDomains: []string{"test.com"},
		Num:            10,
		Start:          5,
	}

	jsonBytes, err := json.Marshal(request)
	require.NoError(t, err)

	// Verify snake_case field name for exclude_domains
	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"q"`)
	assert.Contains(t, jsonStr, `"location"`)
	assert.Contains(t, jsonStr, `"hl"`)
	assert.Contains(t, jsonStr, `"gl"`)
	assert.Contains(t, jsonStr, `"exclude_domains"`)
	assert.Contains(t, jsonStr, `"num"`)
	assert.Contains(t, jsonStr, `"start"`)

	// Verify camelCase is NOT used
	assert.NotContains(t, jsonStr, `"excludeDomains"`)
	assert.NotContains(t, jsonStr, `"ExcludeDomains"`)
}

// TestErrorResponse_JSONFieldName tests that JSON field name is correct
func TestErrorResponse_JSONFieldName(t *testing.T) {
	response := ErrorResponse{Error: "test error"}

	jsonBytes, err := json.Marshal(response)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, `"error"`)
	assert.NotContains(t, jsonStr, `"Error"`)
}

// TestSearchRequest_RoundTrip tests full round-trip marshaling/unmarshaling
func TestSearchRequest_RoundTrip(t *testing.T) {
	original := SearchRequest{
		Q:              "escritório de contabilidade em recife",
		Location:       "Recife, Pernambuco, Brazil",
		Hl:             "pt-br",
		Gl:             "br",
		ExcludeDomains: []string{"instagram.com", "linkedin.com", "facebook.com"},
		Num:            50,
		Start:          100,
	}

	// Marshal
	jsonBytes, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var decoded SearchRequest
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Q, decoded.Q)
	assert.Equal(t, original.Location, decoded.Location)
	assert.Equal(t, original.Hl, decoded.Hl)
	assert.Equal(t, original.Gl, decoded.Gl)
	assert.Equal(t, original.ExcludeDomains, decoded.ExcludeDomains)
	assert.Equal(t, original.Num, decoded.Num)
	assert.Equal(t, original.Start, decoded.Start)
}

// TestErrorResponse_RoundTrip tests full round-trip for ErrorResponse
func TestErrorResponse_RoundTrip(t *testing.T) {
	original := ErrorResponse{
		Error: "Key: 'SearchRequest.Q' Error:Field validation for 'Q' failed on the 'required' tag",
	}

	// Marshal
	jsonBytes, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var decoded ErrorResponse
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Error, decoded.Error)
}
