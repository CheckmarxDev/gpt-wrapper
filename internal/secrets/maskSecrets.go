package secrets

import (
	_ "embed"
	"encoding/json"
	"fmt"
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
	Regex       string      `json:"regex"`
	Entropies   []Entropy   `json:"entropies"`
	Multiline   Multiline   `json:"multiline"`
	AllowRules  []AllowRule `json:"allowRules"`
	SpecialMask string      `json:"specialMask"`
}

type SecretRules struct {
	Rules      []SecretRule `json:"rules"`
	AllowRules []AllowRule  `json:"allowRules"`
}

type SecretRegex struct {
	QueryName   string
	Regex       *regexp.Regexp
	Multiline   Multiline
	Entropies   []Entropy
	AllowRules  []*regexp.Regexp
	SpecialMask *regexp.Regexp
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
		specialMask := regexStruct.SpecialMask
		var allowRules []*regexp.Regexp

		for _, rule := range regexStruct.AllowRules {
			compiledRule, err := regexp.Compile(rule.Regex)
			if err == nil {
				allowRules = append(allowRules, compiledRule)
			}
		}

		var specialMaskCompiled *regexp.Regexp
		if specialMask == "" {
			specialMaskCompiled = nil
		} else {
			specialMaskCompiled, _ = regexp.Compile(specialMask)
		}
		regexCompiled, _ := regexp.Compile(regex)

		secretRegex := &SecretRegex{
			QueryName:   regexStruct.Name,
			Regex:       regexCompiled,
			AllowRules:  allowRules,
			Multiline:   regexStruct.Multiline,
			Entropies:   regexStruct.Entropies,
			SpecialMask: specialMaskCompiled,
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
func getLineNumber(str string, index int) int {
	lineNumber := 1
	for i := 0; i < index; i++ {
		if str[i] == '\n' && i != index-1 {
			lineNumber++
		}
	}
	return lineNumber
}

func getLines(str string, firstLine int, lastLine int) string {
	lineNumber := 1
	var returnLines []byte
	for i := 0; i < len(str) && lineNumber <= lastLine; i++ {
		if lineNumber >= firstLine && (str[i] != '\n' || lineNumber < lastLine) {
			returnLines = append(returnLines, str[i])
		}
		if str[i] == '\n' {
			lineNumber++
		}

	}
	return string(returnLines)
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

// ReplaceMatches If matches between the regex and the file content, then replace the match with the string "<masked>"
func ReplaceMatches(fileName string, result string, regexs []SecretRegex, allowRegexes []*regexp.Regexp) (string, []Result, []maskedSecret.MaskedSecret) {
	var results []Result
	var maskedSecrets []maskedSecret.MaskedSecret
	var multilineRegexes []SecretRegex

	lines := strings.Split(strings.ReplaceAll(result, "\r\n", "\n"), "\n")
	// Replace matches
	for _, re := range regexs {
		if re.Multiline.DetectLineGroup != 0 {
			multilineRegexes = append(multilineRegexes, re)
			continue
		}
		for index, line := range lines {
			maskedSecretElement := maskedSecret.MaskedSecret{}
			lines[index] = re.Regex.ReplaceAllStringFunc(line, func(match string) string {
				for _, allowRule := range append(re.AllowRules, allowRegexes...) {
					if allowRule.FindString(line) != "" {
						return match
					}
				}

				groups := re.Regex.FindAllStringSubmatch(result, -1)
				for _, entropy := range re.Entropies {
					if len(groups) < entropy.Group {
						if ok, _ := CheckEntropyInterval(entropy, groups[0][entropy.Group]); !ok {
							return match
						}
					}
				}

				startOfMatch := ""
				if re.SpecialMask != nil {
					startOfMatch = re.SpecialMask.FindString(line)
					// Add the masked string to return
					maskedSecretElement.Secret = line //line[len(startOfMatch):]
					maskedSecrets = append(maskedSecrets, maskedSecretElement)
				}
				maskedSecret := fmt.Sprintf("%s<masked>", startOfMatch)
				maskedSecretElement.Masked = maskedSecret
				results = append(results, Result{QueryName: "Passwords And Secrets - " + re.QueryName, Line: index + 1, FileName: fileName, Severity: "HIGH"})
				return maskedSecret
			})
		}
	}
	result = strings.Join(lines[:], "\n")
	for _, re := range multilineRegexes {
		// Find all matches of the regular expression in the string
		groups := re.Regex.FindStringSubmatchIndex(result)

		// Iterate over each match
		for groups != nil {
			maskedSecretElement := maskedSecret.MaskedSecret{}
			firstLine := getLineNumber(result, groups[0])
			lastLine := getLineNumber(result, groups[1])
			fullContext := getLines(result, firstLine, lastLine)
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
			matchString := result[groups[0]:groups[1]]

			if len(groups) <= re.Multiline.DetectLineGroup*2 {
				groups = nil
				continue
			}

			stringToMask := result[groups[re.Multiline.DetectLineGroup*2]:groups[re.Multiline.DetectLineGroup*2+1]]
			lineOfSecret := getLineNumber(result, groups[re.Multiline.DetectLineGroup*2])

			startOfMatch := ""
			if re.SpecialMask != nil {
				partOfMatches := re.SpecialMask.FindAllStringIndex(stringToMask, -1)
				if len(partOfMatches) != 0 {
					partOfMatch := partOfMatches[len(partOfMatches)-1]
					startOfMatch = stringToMask[0:partOfMatch[1]]
				}
			}
			maskedSecret := fmt.Sprintf("%s<masked>", startOfMatch)



			results = append(results, Result{QueryName: "Passwords And Secrets - " + re.QueryName, Line: lineOfSecret, FileName: fileName, Severity: "HIGH"})

			maskedMatchString := strings.Replace(matchString, stringToMask, maskedSecret, 1)

			// Add the masked string to return
			maskedSecretElement.Masked = maskedSecret
			maskedSecretElement.Secret = matchString
			maskedSecrets = append(maskedSecrets, maskedSecretElement)

			result = strings.Replace(result, matchString, maskedMatchString, 1)

			groups = re.Regex.FindStringSubmatchIndex(result)
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
