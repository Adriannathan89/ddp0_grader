# Grader runner dan queue

Package `pkg/runner` menjalankan source Python dalam container Docker yang singkat dan terisolasi. Package `pkg/queue` menyediakan antrean Redis Streams dengan batas 10 worker yang dapat dijalankan oleh caller.

## Wiring ke main module

Karena Redis sudah diinisialisasi oleh `app/config`, expose dan gunakan client yang sama:

```go
package main

import (
    "context"
    "log"
    "time"

    "ddp0_grader/app/config"
    "ddp0_grader/pkg/queue"
    "ddp0_grader/pkg/runner"
)

func main() {
    config.InitConfig()

    ctx := context.Background()
    r := runner.New(runner.Config{
        Image:           "python:3.12-slim",
        OutputLimit:     1 << 20,
        DefaultTime:     2 * time.Second,
        DefaultMemoryMB: 256,
    })
    q, err := queue.NewWithClient(config.RedisClient, queue.Config{
        Stream:   "grader:jobs",
        Group:    "grader-workers",
        Consumer: "api-runner",
    })
    if err != nil { log.Fatal(err) }

    go func() {
        if err := q.WorkN(ctx, 10, func(ctx context.Context, job queue.Job) error {
        results, err := r.Run(ctx, &job.Submission, &job.Problem, job.TestCases)
        if err != nil { return err }

        // Simpan results ke database di sini.
        // Contoh: persistSubmissionResult(job.Submission.ID, results)
        _ = results
        return nil
        }); err != nil {
            log.Printf("grader workers stopped: %v", err)
        }
    }()

    // Jalankan HTTP server setelah worker dimulai.
    // r := gin.Default()
    // ... register routes ...
    // log.Fatal(r.Run(":" + config.GetEnv("PORT")))
}
```

`main()` hanya dieksekusi sekali untuk setiap process. `config.InitConfig()` juga cukup dipanggil sekali. Dalam satu process, `config.RedisClient` dapat dibagikan ke semua worker karena client `go-redis` aman digunakan concurrently.

`WorkN(ctx, 10, handler)` bersifat blocking. Jika API dan worker berada dalam process yang sama, jalankan dengan `go q.WorkN(...)` seperti contoh di atas. Jika worker berada dalam binary/process terpisah, panggil `config.InitConfig()` sekali di `main()` worker tersebut lalu jalankan `q.WorkN(...)` secara langsung tanpa `go`.

Untuk memasukkan submission ke queue dari handler API:

```go
func enqueueSubmission(ctx context.Context, q *queue.Queue, submission models.Submission, problem models.Problem, testCases []models.TestCase) error {
    _, err := q.Enqueue(ctx, queue.Job{
        ID:         submission.ID,
        Submission: submission,
        Problem:    problem,
        TestCases:  testCases,
    })
    return err
}
```

Pastikan hanya satu `queue.Queue` yang memanggil `Close()` untuk setiap Redis client yang memang dibuat oleh queue. Jika menggunakan `NewWithClient`, client dimiliki oleh `app/config`, sehingga lifecycle-nya dikelola oleh aplikasi dan tidak perlu ditutup oleh queue.

## Prasyarat

- Docker daemon aktif.
- Image Python, misalnya `python:3.12-slim`, tersedia di host runner.
- Redis aktif.

## Seed problem Hello World

Seeder membuat satu problem dengan ID `hello-world` dan dua testcase. Seeder bersifat idempotent sehingga dapat dijalankan berulang kali:

```bash
go run ./cmd/seed
```

Seeder hanya membutuhkan koneksi PostgreSQL dari environment database dan tidak membutuhkan Redis.

## Menjalankan runner

```go
r := runner.New(runner.Config{
    Image: "python:3.12-slim",
    OutputLimit: 1 << 20,
})
results, err := r.Run(ctx, &submission, &problem, testCases)
```

Setiap testcase mendapatkan container sendiri dengan `--network none`, filesystem read-only, batas memory, CPU, PID, timeout, dan batas ukuran stdout. Output dibandingkan sebagai token whitespace-insensitive dengan `TestCase.Output`. `stderr` hanya untuk diagnosis.

## Menjalankan Redis queue

```go
q := queue.New(queue.Config{
    Addr: "localhost:6379",
    Group: "grader-workers",
    Consumer: "worker-1",
})
defer q.Close()

err := q.WorkN(ctx, 10, func(ctx context.Context, job queue.Job) error {
    _, err := r.Run(ctx, &job.Submission, &job.Problem, job.TestCases)
    return err
})
```

Enqueue submission:

```go
_, err := q.Enqueue(ctx, queue.Job{
    ID: submission.ID,
    Submission: submission,
    Problem: problem,
    TestCases: testCases,
})
```

`WorkN(ctx, 10, handler)` menjalankan 10 worker secara concurrent. Job lain menunggu di Redis. Message baru di-ACK setelah handler sukses; jika handler error, message tetap pending dan dapat diproses ulang setelah recovery mechanism ditambahkan.

`Problem.TimeLimit` dianggap dalam milidetik dan `Problem.MemoryLimit` dalam MB.

Untuk produksi, sebaiknya payload queue hanya berisi ID submission. Source code dan testcase besar disimpan di database/object storage agar Redis tidak cepat penuh.

## Batas memory

Jika `Problem.MemoryLimit` adalah 256 MB dan ada 10 container aktif, batas maksimum proses peserta adalah sekitar 2.560 MB, ditambah Redis, Go process, Docker, dan OS. Host 4 GB adalah minimum praktis; 8 GB lebih nyaman.

## Upload submission

Endpoint yang sudah di-wire ke `cmd/api`:

```bash
curl -X POST http://localhost:8080/submissions/grade \
  -F "problem_id=problem-1" \
  -F "user_id=user-1" \
  -F "file=@solution.py"
```

Response berhasil adalah `202 Accepted`:

```json
{
  "submission_id": "...",
  "status": "queued"
}
```

Field `file` wajib berekstensi `.py` dan dibatasi maksimal 1 MiB. Endpoint hanya memasukkan job ke Redis; hasil akhir disimpan oleh worker ke `submissions` dan `test_case_results`.
