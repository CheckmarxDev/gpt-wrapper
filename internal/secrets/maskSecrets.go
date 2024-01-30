package secrets

import (
	_ "embed"
	"encoding/json"
	"math"
	"regexp"
	"strings"

	"github.com/checkmarxDev/gpt-wrapper/pkg/maskedSecret"
)

const (
	Base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	HexChars    = "1234567890abcdefABCDEF"
)

//go:embed regex_rules.json
var regexRules []byte

type Entropy struct {
	Group int     `json:"group"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

type Multiline struct {
	DetectLineGroup int `json:"detectLineGroup"`
}

type AllowRule struct {
	Description string `json:"description"`
	Regex       string `json:"regex"`
}

type SecretRule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Multiline   bool        `json:"multiline"`
	Regex       string      `json:"regex"`
	Entropies   []Entropy   `json:"entropies"`
	AllowRules  []AllowRule `json:"allowRules"`
	GroupToMask int         `json:"groupMask"`
}

type SecretRules struct {
	Rules      []SecretRule `json:"rules"`
	AllowRules []AllowRule  `json:"allowRules"`
}

type SecretRegex struct {
	QueryName   string
	Regex       *regexp.Regexp
	RegexStr    string
	Multiline   bool
	GroupToMask int
	Entropies   []Entropy
	AllowRules  []*regexp.Regexp
}

type Result struct {
	QueryName string `json:"queryName"`
	Line      int    `json:"line"`
	FileName  string `json:"fileName"`
	Severity  string `json:"severity"`
}

// LoadRegexps Load custom regexps
func LoadRegexps() ([]SecretRegex, []*regexp.Regexp, error) {
	var allowRulesRegexes []*regexp.Regexp

	var secretRules SecretRules
	err := json.Unmarshal(regexRules, &secretRules)
	if err != nil {
		return nil, nil, err
	}

	var regexes []SecretRegex
	for _, regexStruct := range secretRules.Rules {
		regex := regexStruct.Regex
		var allowRules []*regexp.Regexp

		for _, rule := range regexStruct.AllowRules {
			compiledRule, err := regexp.Compile(rule.Regex)
			if err == nil {
				allowRules = append(allowRules, compiledRule)
			}
		}

		regexCompiled, _ := regexp.Compile(regex)

		secretRegex := &SecretRegex{
			QueryName:   regexStruct.Name,
			Regex:       regexCompiled,
			AllowRules:  allowRules,
			Multiline:   regexStruct.Multiline,
			Entropies:   regexStruct.Entropies,
			GroupToMask: regexStruct.GroupToMask,
			RegexStr:    regexStruct.Regex,
		}
		regexes = append(regexes, *secretRegex)
	}

	for _, allowRegex := range secretRules.AllowRules {
		compiledRule, err := regexp.Compile(allowRegex.Regex)
		if err == nil {
			allowRulesRegexes = append(allowRulesRegexes, compiledRule)
		}
	}

	return regexes, allowRulesRegexes, nil
}

// getLineNumber calculates the line number based on the match index
func getLineNumber(text, portion string) int {
	index := strings.Index(text, portion) + 1
	lineNumber := 1
	for i := 0; i < index; i++ {
		if text[i] == '\n' && i != index-1 {
			lineNumber++
		}
	}
	return lineNumber
}

// CheckEntropyInterval - verifies if a given token's entropy is within expected bounds
func CheckEntropyInterval(entropy Entropy, token string) (isEntropyInInterval bool, entropyLevel float64) {
	base64Entropy := calculateEntropy(token, Base64Chars)
	hexEntropy := calculateEntropy(token, HexChars)
	highestEntropy := math.Max(base64Entropy, hexEntropy)
	if insideInterval(entropy, base64Entropy) || insideInterval(entropy, hexEntropy) {
		return true, highestEntropy
	}
	return false, highestEntropy
}

func insideInterval(entropy Entropy, floatEntropy float64) bool {
	return floatEntropy >= entropy.Min && floatEntropy <= entropy.Max
}

// calculateEntropy - calculates the entropy of a string based on the Shannon formula
func calculateEntropy(token, charSet string) float64 {
	if token == "" {
		return 0
	}
	charMap := map[rune]float64{}
	for _, char := range token {
		if strings.Contains(charSet, string(char)) {
			charMap[char]++
		}
	}

	var freq float64
	length := float64(len(token))
	for _, count := range charMap {
		freq += count * math.Log2(count)
	}

	return math.Log2(length) - freq/length
}

// maskRegexByMatchGroup masks a content using a regex and the group to be masked
func maskRegexByMatchGroup(groupToMask int, matchContent string, query *SecretRegex) string {
	query.Regex = regexp.MustCompile(".*" + query.RegexStr) // add .* to match the last appearance
	groups := query.Regex.FindAllStringSubmatch(matchContent, -1)
	lastMatch := groups[len(groups)-1]
	if len(lastMatch) < groupToMask {
		return matchContent
	}
	return strings.Replace(matchContent, lastMatch[groupToMask], "<masked>", 1)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IsAllowRule check if string matches any of the allow rules for the secret queries
func IsAllowRule(s string, query *SecretRegex, allowRules []*regexp.Regexp) bool {
	query.Regex = regexp.MustCompile(query.RegexStr)
	regexMatch := query.Regex.FindStringIndex(s)
	if regexMatch != nil {
		allowRuleMatches := AllowRuleMatches(s, append(query.AllowRules, allowRules...))

		for _, allowMatch := range allowRuleMatches {
			allowStart, allowEnd := allowMatch[0], allowMatch[1]
			regexStart, regexEnd := regexMatch[0], regexMatch[1]

			if (allowStart <= regexEnd && allowStart >= regexStart) || (regexStart <= allowEnd && regexStart >= allowStart) {
				return true
			}
		}
	}

	return false
}

// AllowRuleMatches return all the allow rules matches for the secret queries
func AllowRuleMatches(s string, allowRules []*regexp.Regexp) [][]int {
	allowRuleMatches := [][]int{}
	for i := range allowRules {
		res := allowRules[i].FindAllStringIndex(s, -1)
		allowRuleMatches = append(allowRuleMatches, res...)
	}
	return allowRuleMatches
}

// ReplaceMatches If matches between the regex and the file content, then replace the match with the string "<masked>"
func ReplaceMatches(fileName string, result string, regexs []SecretRegex, allowRegexes []*regexp.Regexp) (string, []Result, []maskedSecret.MaskedSecret) {
	var results []Result
	var maskedSecrets []maskedSecret.MaskedSecret
	var multilineRegexes []SecretRegex

	lines := strings.Split(strings.ReplaceAll(result, "\r\n", "\n"), "\n")
	// Replace matches
	for _, re := range regexs {
		if re.Multiline {
			multilineRegexes = append(multilineRegexes, re)
			continue
		}
		for index, line := range lines {
			originalLine := lines[index]
			lines[index] = re.Regex.ReplaceAllStringFunc(line, func(match string) string {
				if IsAllowRule(line, &re, append(re.AllowRules, allowRegexes...)) {
					return match
				}
				re.Regex = regexp.MustCompile(re.RegexStr)
				groups := re.Regex.FindAllStringSubmatch(result, -1)
				for _, entropy := range re.Entropies {
					if len(groups) < entropy.Group {
						if ok, _ := CheckEntropyInterval(entropy, groups[0][entropy.Group]); !ok {
							return match
						}
					}
				}

				maskedSecret := maskRegexByMatchGroup(re.GroupToMask, match, &re)
				results = append(results, Result{QueryName: "Passwords And Secrets - " + re.QueryName, Line: index + 1, FileName: fileName, Severity: "HIGH"})
				return maskedSecret
			})
			if originalLine != lines[index] {
				// Add the masked string to return
				maskedSecretElement := maskedSecret.MaskedSecret{}
				maskedSecretElement.Secret = originalLine
				maskedSecretElement.Masked = lines[index]
				maskedSecretElement.Line = index
				maskedSecrets = append(maskedSecrets, maskedSecretElement)
			}
		}
	}
	result = strings.Join(lines[:], "\n")
	for _, re := range multilineRegexes {
		// Find all matches of the regular expression in the string
		re.Regex = regexp.MustCompile(re.RegexStr)
		groups := re.Regex.FindStringSubmatch(result)

		// Iterate over each match
		for groups != nil {
			maskedSecretElement := maskedSecret.MaskedSecret{}
			fullContext := groups[0]
			allowed := false

			for _, allowRule := range append(re.AllowRules, allowRegexes...) {
				if allowRule.FindString(fullContext) != "" {
					allowed = true
					break
				}
			}
			if allowed { // Allowed by the allowRules of the regex
				groups = nil
				continue
			}

			// Extract the matched substring
			matchString := groups[0]

			lineOfSecret := getLineNumber(result, groups[re.GroupToMask])

			maskedMatchString := maskRegexByMatchGroup(re.GroupToMask, matchString, &re)
			results = append(results, Result{QueryName: "Passwords And Secrets - " + re.QueryName, Line: lineOfSecret, FileName: fileName, Severity: "HIGH"})

			// Add the masked string to return
			maskedSecretElement.Masked = maskedMatchString
			maskedSecretElement.Secret = matchString
			maskedSecretElement.Line = lineOfSecret

			maskedSecrets = append(maskedSecrets, maskedSecretElement)

			result = strings.Replace(result, matchString, maskedMatchString, 1)

			groups = re.Regex.FindStringSubmatch(result)
		}
	}
	return result, results, maskedSecrets
}

func MaskSecrets(fileContent string) (string, []maskedSecret.MaskedSecret, error) {
	// Load regexps
	rs, allowRs, err := LoadRegexps()
	if err != nil {
		return "", nil, err
	}

	maskedResult, _, maskedSecrets := ReplaceMatches("", fileContent, rs, allowRs)
	return maskedResult, maskedSecrets, nil
}
