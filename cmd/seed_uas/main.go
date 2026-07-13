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
	id    string
	input string
}

func main() {
	config.InitDatabase()
	problem := models.Problem{
		ID:          problemID,
		Title:       "Kita Balas di UAS",
		Description: description,
		TimeLimit:   1000,
		MemoryLimit: 128,
	}

	tests := []testData{
		{"kita-balas-di-uas-01", "4\nTugasMingguan 97 10\nLab 94.5 20\nKuis 95.7 5\nUTS 67 30\n"},
		{"kita-balas-di-uas-02", "5\nLab 90 0\nQuiz 66.6 20\nTugasCoding 88.8 10\nKeaktifan 100 10\nUTS 77.7 30\n"},
		{"kita-balas-di-uas-03", "1\nTugas 50 0\n"},
		{"kita-balas-di-uas-04", "1\nUTS 90 100\n"},
		{"kita-balas-di-uas-05", "2\nTugas 80 60\nUTS 80 50\n"},
		{"kita-balas-di-uas-06", "1\nUTS 70 50\n"},
		{"kita-balas-di-uas-07", "2\nTugas 100 25\nUTS 100 15\n"},
		{"kita-balas-di-uas-08", "20\nC01 51.5 4\nC02 62.5 4\nC03 73.5 4\nC04 84.5 4\nC05 95.5 4\nC06 56.5 4\nC07 67.5 4\nC08 78.5 4\nC09 89.5 4\nC10 99.5 4\nC11 52.25 4\nC12 63.25 4\nC13 74.25 4\nC14 85.25 4\nC15 96.25 4\nC16 57.75 4\nC17 68.75 4\nC18 79.75 4\nC19 90.75 4\nC20 100 4\n"},
		{"kita-balas-di-uas-09", "3\nLab 80.25 25.5\nKuis 75.75 24.5\nUTS 90 25\n"},
		{"kita-balas-di-uas-10", "3\nTugas 100 30\nKuis 95 25\nUTS 96 20\n"},
	}

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
				IsHidden:  true,
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
