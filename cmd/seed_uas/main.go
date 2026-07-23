package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"ddp0_grader/app/config"
	"ddp0_grader/app/models"

	"gorm.io/gorm"
)

const problemID = "kita-balas-di-uas"

const description = `Diberikan N komponen penilaian selain UAS. Setiap komponen memiliki
nilai dan bobot terhadap nilai akhir. Bobot UAS adalah sisa bobot hingga 100%:

  bobot UAS = 100 - jumlah bobot komponen yang diberikan

Hitung nilai minimal UAS agar nilai akhir minimal 85.00. Jika nilai minimal UAS
lebih dari 100, keluarkan "Tidak bisa!". Jika hasilnya negatif, keluarkan 0.00.
Nilai akhir dan bobot dihitung sebagai persentase, lalu jawaban dibulatkan dan
ditampilkan dengan tepat dua angka di belakang koma.

Catatan seed: contoh kedua pada statement awal menghasilkan 73.73 berdasarkan
aturan bobot UAS di atas; expected output "Tidak bisa!" pada contoh tersebut
tidak konsisten dengan data input.`

type testData struct {
	id       string
	input    string
	isHidden bool
}

func main() {
	config.InitDatabase()
	problem := models.Problem{
		ID:          problemID,
		Title:       "Kita Balas di UAS",
		Description: description,
		Author:      "system",
		Tag:         models.TagMath,
		Difficulty:  models.DifficultyEasy,
		TimeLimit:   1000,
		MemoryLimit: 64,
	}

	tests := seedTests()

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", problem.ID).Assign(problem).FirstOrCreate(&problem).Error; err != nil {
			return err
		}
		for _, test := range tests {
			tc := models.TestCase{
				ID:        test.id,
				ProblemID: problemID,
				Input:     test.input,
				Output:    expected(test.input),
				IsHidden:  test.isHidden,
			}
			if err := tx.Where("id = ?", tc.ID).Assign(tc).FirstOrCreate(&tc).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Fatalf("seed UAS failed: %v", err)
	}

	log.Printf("seeded problem %q with %d test cases", problemID, len(tests))
}

func seedTests() []testData {
	return []testData{
		// Public testcase: sample that contestants can use to verify formatting.
		{"kita-balas-di-uas-01", "4\nTugasMingguan 97 10\nLab 94.5 20\nKuis 95.7 5\nUTS 67 30\n", false},
		{"kita-balas-di-uas-02", "5\nLab 90 0\nQuiz 66.6 20\nTugasCoding 88.8 10\nKeaktifan 100 10\nUTS 77.7 30\n", true},
		{"kita-balas-di-uas-03", "1\nTugas 50 0\n", true},
		{"kita-balas-di-uas-04", "1\nUTS 90 100\n", true},
		{"kita-balas-di-uas-05", "2\nTugas 80 60\nUTS 80 50\n", true},
		{"kita-balas-di-uas-06", "1\nUTS 70 50\n", true},
		{"kita-balas-di-uas-07", "2\nTugas 100 25\nUTS 100 15\n", true},
		{"kita-balas-di-uas-08", "20\nC01 51.5 4\nC02 62.5 4\nC03 73.5 4\nC04 84.5 4\nC05 95.5 4\nC06 56.5 4\nC07 67.5 4\nC08 78.5 4\nC09 89.5 4\nC10 99.5 4\nC11 52.25 4\nC12 63.25 4\nC13 74.25 4\nC14 85.25 4\nC15 96.25 4\nC16 57.75 4\nC17 68.75 4\nC18 79.75 4\nC19 90.75 4\nC20 100 4\n", true},
		{"kita-balas-di-uas-09", "3\nLab 80.25 25.5\nKuis 75.75 24.5\nUTS 90 25\n", true},
		{"kita-balas-di-uas-10", "3\nTugas 100 30\nKuis 95 25\nUTS 96 20\n", true},
		{"kita-balas-di-uas-11", "1\nTugas 85 50\n", true},
		{"kita-balas-di-uas-12", "1\nKuis 0 1\n", true},
		{"kita-balas-di-uas-13", "2\nTugas 0 10\nUTS 0 20\n", true},
		{"kita-balas-di-uas-14", "2\nTugas 100 40\nUTS 0 20\n", true},
		{"kita-balas-di-uas-15", "3\nTugas 90 10\nKuis 80 10\nUTS 70 10\n", true},
		{"kita-balas-di-uas-16", "4\nA 100 5\nB 100 5\nC 100 5\nD 100 5\n", true},
		{"kita-balas-di-uas-17", "5\nA 72.5 12.5\nB 68.75 17.5\nC 91.25 10\nD 83.33 5\nE 77.77 20\n", true},
		{"kita-balas-di-uas-18", "6\nA 60 10\nB 65 10\nC 70 10\nD 75 10\nE 80 10\nF 85 10\n", true},
		{"kita-balas-di-uas-19", "10\nA 100 1\nB 90 2\nC 80 3\nD 70 4\nE 60 5\nF 50 6\nG 40 7\nH 30 8\nI 20 9\nJ 10 10\n", true},
		{"kita-balas-di-uas-20", "20\nA01 100 3\nA02 95 3\nA03 90 3\nA04 85 3\nA05 80 3\nA06 75 3\nA07 70 3\nA08 65 3\nA09 60 3\nA10 55 3\nA11 50 3\nA12 45 3\nA13 40 3\nA14 35 3\nA15 30 3\nA16 25 3\nA17 20 3\nA18 15 3\nA19 10 3\nA20 5 3\n", true},
		{"kita-balas-di-uas-21", "2\nProyek 100 70\nUTS 100 20\n", true},
		{"kita-balas-di-uas-22", "2\nProyek 50 70\nUTS 60 20\n", true},
		{"kita-balas-di-uas-23", "3\nTugas 99.99 33.33\nKuis 88.88 33.33\nUTS 77.77 33.33\n", true},
		{"kita-balas-di-uas-24", "4\nA 85 20\nB 85 20\nC 85 20\nD 85 20\n", true},
		{"kita-balas-di-uas-25", "4\nA 84.99 20\nB 84.99 20\nC 84.99 20\nD 84.99 20\n", true},
		{"kita-balas-di-uas-26", "1\nUTS 84.995 99\n", true},
		{"kita-balas-di-uas-27", "3\nTugas 10 25\nKuis 20 25\nUTS 30 25\n", true},
		{"kita-balas-di-uas-28", "5\nA 100 0\nB 100 0\nC 100 0\nD 100 0\nE 100 0\n", true},
		{"kita-balas-di-uas-29", "2\nTugas 100 49.99\nUTS 0 49.99\n", true},
		{"kita-balas-di-uas-30", "7\nA 91 7\nB 82 8\nC 73 9\nD 64 10\nE 55 11\nF 46 12\nG 37 13\n", true},
	}
}

func expected(input string) string {
	fields := strings.Fields(input)
	n, _ := strconv.Atoi(fields[0])
	total := 0.0
	weight := 0.0
	pos := 1
	for i := 0; i < n; i++ {
		pos++ // component name
		mark, _ := strconv.ParseFloat(fields[pos], 64)
		pos++
		w, _ := strconv.ParseFloat(fields[pos], 64)
		pos++
		total += mark * w / 100
		weight += w
	}
	uasWeight := 100 - weight
	if uasWeight <= 0 {
		return "Tidak bisa!\n"
	}
	minimum := (85 - total) * 100 / uasWeight
	if minimum > 100 {
		return "Tidak bisa!\n"
	}
	if minimum < 0 {
		minimum = 0
	}
	return fmt.Sprintf("%.2f\n", minimum)
}
