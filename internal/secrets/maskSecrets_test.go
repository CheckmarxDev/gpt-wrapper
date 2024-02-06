package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/rs/zerolog/log"
)

func TestSecretsDetection(t *testing.T) {
	expectedResultsPath := "test/positive_expected_result.json"
	expectedResults, err := os.ReadFile(expectedResultsPath)
	if err != nil {
		t.Fatal(err)
	}

	// Load expected results
	var expectedResultList []Result
	err = json.Unmarshal(expectedResults, &expectedResultList)
	if err != nil {
		t.Fatal(err)
	}

	// Load regexs
	rs, allowrs, err := LoadRegexps()
	if err != nil {
		t.Fatal(err)
	}

	// Load test files
	folderPath := "test"
	dir, err := os.ReadDir(folderPath)
	if err != nil {
		t.Fatal(err)
	}

	var multiLineQueries []string
	var results []Result
	for _, entry := range dir {
		if !entry.IsDir() && entry.Name() != "positive_expected_result.json" {
			filePath := filepath.Join(folderPath, entry.Name())
			results = append(results, processFile(t, filePath, rs, allowrs)...)
		}
	}

	for _, ree := range rs {
		if ree.Multiline {
			multiLineQueries = append(multiLineQueries, ree.QueryName)
		}
	}

	if !compareExpectedWithActual(expectedResultList, results, multiLineQueries) {
		t.Fatalf("Failed comparing expected results with actual results: %v\n", results)
	}
}

func processFile(t *testing.T, path string, rs []SecretRegex, allowrs []*regexp.Regexp) []Result {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	_, res, _ := ReplaceMatches(path, string(fileContent), rs, allowrs)

	return res

}

func diffActualExpectedVulnerabilities(actual, expected []Result) []string {
	m := make(map[string]bool)
	diff := make([]string, 0)
	for i := range expected {
		m[expected[i].QueryName+":"+filepath.Base(expected[i].FileName)+":"+strconv.Itoa(expected[i].Line)] = true
	}
	for i := range actual {
		if _, ok := m[actual[i].QueryName+":"+filepath.Base(actual[i].FileName)+":"+strconv.Itoa(actual[i].Line)]; !ok {
			diff = append(diff, actual[i].FileName+":"+strconv.Itoa(actual[i].Line))
		}
	}

	return diff
}

func compareExpectedWithActual(expected, actual []Result, multiLineQueries []string) bool {
	if len(expected) != len(actual) {
		log.Error().Msgf(
			"Count of actual issues and expected vulnerabilities doesn't match\n -- \n"+
				"not present in expected and present in actual: %v\n"+
				"not present in actual and present in expected: %v\n",
			diffActualExpectedVulnerabilities(actual, expected),
			diffActualExpectedVulnerabilities(expected, actual))
		return false
	}

	for _, resExp := range expected {
		found := false
		multiLine := false
		for _, multiLineQuery := range multiLineQueries {
			if strings.HasSuffix(resExp.QueryName, multiLineQuery) {
				multiLine = true
			}
		}
		for _, resAct := range actual {
			if strings.HasSuffix(resAct.FileName, resExp.FileName) && resAct.QueryName == resExp.QueryName && resAct.Severity == resExp.Severity && (resAct.Line == resExp.Line || multiLine) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
