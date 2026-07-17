package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strings"

	"ddp0_grader/app/config"
	"ddp0_grader/app/models"

	"gorm.io/gorm"
)

type problemDefinition struct {
	Title       string   `json:"title"`
	Tag         string   `json:"tag"`
	Description string   `json:"description"`
	Solution    []string `json:"solution"`
}

type generatedTestCase struct {
	Input  string
	Output string
}

func main() {
	config.InitDatabase()

	definitions, err := loadDefinitions()
	if err != nil {
		log.Fatal(err)
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		for _, definition := range definitions {
			if err := seedProblem(tx, definition); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Fatalf("seed problems failed: %v", err)
	}

	log.Printf("seeded %d problems with 30 test cases each", len(definitions))
}

func loadDefinitions() ([]problemDefinition, error) {
	paths := []string{"cmd/seeding_problem/problem.json", "problem.json"}
	var data []byte
	var err error
	for _, path := range paths {
		data, err = os.ReadFile(path)
		if err == nil {
			var definitions []problemDefinition
			if err := json.Unmarshal(data, &definitions); err != nil {
				return nil, fmt.Errorf("decode %s: %w", path, err)
			}
			return definitions, nil
		}
	}
	return nil, fmt.Errorf("read problem.json: %w", err)
}

func seedProblem(tx *gorm.DB, definition problemDefinition) error {
	id := slugify(definition.Title)
	problem := models.Problem{
		ID: id, Title: definition.Title, Description: definition.Description,
		Author: "system", Tag: strings.ToLower(definition.Tag),
		Difficulty: models.DifficultyEasy, TimeLimit: 1000, MemoryLimit: 128,
	}
	if err := tx.Where("id = ?", id).Assign(problem).FirstOrCreate(&problem).Error; err != nil {
		return fmt.Errorf("save problem %s: %w", id, err)
	}

	tests := generateTestCases(definition.Title)
	if len(tests) != 30 {
		return fmt.Errorf("problem %q generated %d testcases, want 30", definition.Title, len(tests))
	}
	if err := tx.Where("problem_id = ?", id).Delete(&models.TestCase{}).Error; err != nil {
		return fmt.Errorf("clear testcases for %s: %w", id, err)
	}
	for i, test := range tests {
		caseID := fmt.Sprintf("%s-%02d", id, i+1)
		row := models.TestCase{ID: caseID, ProblemID: id, Input: test.Input, Output: test.Output, IsHidden: i != 0}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("save testcase %s: %w", caseID, err)
		}
	}
	return nil
}

func generateTestCases(title string) []generatedTestCase {
	switch title {
	case "Antek-Antek Asink":
		return generateFloodCases()
	case "DepeCipher":
		return generateCipherCases()
	case "Zipline Pylon":
		return generateZiplineCases()
	case "Kita Balas di UAS":
		return generateUASCases()
	default:
		panic("no testcase generator for " + title)
	}
}

func generateFloodCases() []generatedTestCase {
	result := make([]generatedTestCase, 0, 30)
	for k := 0; k < 30; k++ {
		n, m := 1+k%10, 1+(k*3)%12
		if k >= 25 {
			// Stress cases: 90k–160k cells, still comfortable for the
			// iterative Python flood-fill under a one-second limit.
			sizes := [][2]int{{200, 200}, {300, 300}, {400, 300}, {300, 400}, {400, 400}}
			n, m = sizes[k-25][0], sizes[k-25][1]
		}
		lines := make([]string, n)
		grid := make([][]byte, n)
		for i := range grid {
			grid[i] = make([]byte, m)
			for j := range grid[i] {
				if (i*7+j*11+k)%5 == 0 {
					grid[i][j] = '#'
				} else {
					grid[i][j] = '.'
				}
			}
			lines[i] = string(grid[i])
		}
		// Keep a few boundary cases explicit: empty land and one connected field.
		if k == 0 {
			lines[0] = "#"
		}
		if k == 1 {
			for i := range lines {
				lines[i] = strings.Repeat(".", m)
			}
		}
		if k == 29 {
			for i := range lines {
				lines[i] = strings.Repeat(".", m)
			}
		}
		result = append(result, generatedTestCase{fmt.Sprintf("%d %d\n%s\n", n, m, strings.Join(lines, "\n")), fmt.Sprintf("%d\n", floodCount(lines))})
	}
	return result
}

func floodCount(lines []string) int {
	grid := make([][]byte, len(lines))
	for i := range lines {
		grid[i] = []byte(lines[i])
	}
	count := 0
	for r := range grid {
		for c := range grid[r] {
			if grid[r][c] != '.' {
				continue
			}
			count++
			stack := [][2]int{{r, c}}
			grid[r][c] = '+'
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					nr, nc := p[0]+d[0], p[1]+d[1]
					if nr >= 0 && nr < len(grid) && nc >= 0 && nc < len(grid[nr]) && grid[nr][nc] == '.' {
						grid[nr][nc] = '+'
						stack = append(stack, [2]int{nr, nc})
					}
				}
			}
		}
	}
	return count
}

func generateCipherCases() []generatedTestCase {
	result := make([]generatedTestCase, 0, 30)
	for k := 0; k < 30; k++ {
		s1Length, s2Length := 1+k*7, 3+k*5
		if k >= 24 {
			// The reference solution builds the answer one character at a
			// time, so keep the largest case at 3,000 characters.
			s1Length, s2Length = 1000+(k-24)*400, 1000+(k-24)*500
		}
		s1 := makePatternString(s1Length, 1+k*13)
		s2 := makePatternString(s2Length, 2+k*17)
		result = append(result, generatedTestCase{s1 + "\n" + s2 + "\n", cipher(s1, s2) + "\n"})
	}
	return result
}

func makePatternString(length, shift int) string {
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(byte('a' + (i*19+shift)%26))
	}
	return b.String()
}
func cipher(s1, s2 string) string {
	total := 0
	for _, c := range s2 {
		total += int(c - 'a')
	}
	var b strings.Builder
	for _, c := range s1 {
		b.WriteByte(byte('a' + (int(c-'a')*total)%26))
	}
	return b.String()
}

func generateZiplineCases() []generatedTestCase {
	result := make([]generatedTestCase, 0, 30)
	for k := 0; k < 30; k++ {
		n := 1 + k%15
		if k >= 25 {
			// Stress cases with up to 2,000 pylons; each transition is O(1).
			n = 1000 + (k-25)*250
		}
		lines := make([]string, n)
		x, y := 0.0, 0.0
		lines[0] = fmt.Sprintf("%.2f %.2f", x, y)
		valid := true
		for i := 1; i < n; i++ {
			step := float64(10 + (i+k)%60)
			if k%4 == 0 && i == n/2 {
				step = 81
			}
			x += step
			y += float64((i*3 + k) % 7)
			lines[i] = fmt.Sprintf("%.2f %.2f", x, y)
			if math.Hypot(step, float64((i*3+k)%7)) > 80 {
				valid = false
			}
		}
		out := "Valid\n"
		if !valid {
			out = "Tidak Valid\n"
		}
		result = append(result, generatedTestCase{fmt.Sprintf("%d\n%s\n", n, strings.Join(lines, "\n")), out})
	}
	return result
}

func generateUASCases() []generatedTestCase {
	result := make([]generatedTestCase, 0, 30)
	for k := 0; k < 30; k++ {
		n := 1 + k%20
		if k == 0 {
			n = 4
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%d\n", n)
		total, weight := 0.0, 0.0
		for i := 0; i < n; i++ {
			mark := float64((37*i + 17*k) % 101)
			w := float64((i+k)%8 + 1)
			name := fmt.Sprintf("C%d", i+1)
			if k == 0 {
				name = []string{"TugasMingguan", "Lab", "Kuis", "UTS"}[i]
				mark = []float64{97, 94.5, 95.7, 67}[i]
				w = []float64{10, 20, 5, 30}[i]
			}
			if k == 29 {
				w = 100 / float64(n)
			}
			fmt.Fprintf(&b, "%s %.2f %.2f\n", name, mark, w)
			total += mark * w / 100
			weight += w
		}
		result = append(result, generatedTestCase{b.String(), uasExpected(total, weight)})
	}
	return result
}
func uasExpected(total, weight float64) string {
	remaining := 100 - weight
	if remaining <= 0 {
		if total >= 85 {
			return "0.00\n"
		}
		return "Tidak bisa!\n"
	}
	answer := (85 - total) * 100 / remaining
	if answer > 100 {
		return "Tidak bisa!\n"
	}
	if answer < 0 {
		answer = 0
	}
	return fmt.Sprintf("%.2f\n", answer)
}
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	return strings.Trim(re.ReplaceAllString(s, "-"), "-")
}
