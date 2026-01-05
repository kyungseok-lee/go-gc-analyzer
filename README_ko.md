# Go GC Analyzer

Go 가비지 컬렉션 성능을 모니터링, 분석, 최적화하기 위한 종합적인 GC 분석기입니다.

## 개요

이 라이브러리는 다음 기능을 제공합니다:

- **모니터링**: 설정 가능한 콜백과 함께 실시간 GC 메트릭 수집
- **분석**: GC 성능 패턴 분석 및 병목 지점 식별
- **리포트**: 다양한 형식 지원 (텍스트, JSON, Prometheus)
- **권장사항**: 수집된 데이터 기반 최적화 제안

## 주요 기능

- 📊 실시간 GC 메트릭 수집
- 📈 pause time 백분위수(P95/P99) 포함 종합 GC 분석
- 🔔 성능 이슈 알림 콜백
- 📝 다양한 리포트 형식 (텍스트, JSON, 요약, Prometheus)
- 🏥 헬스 체크 점수 시스템
- 🔄 메모리 트렌드 분석
- 💡 자동 최적화 권장사항 생성

## 설치

```bash
go get github.com/kyungseok-lee/go-gc-analyzer
```

## 빠른 시작

### 기본 사용법

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"
    
    "github.com/kyungseok-lee/go-gc-analyzer/pkg/gcanalyzer"
)

func main() {
    ctx := context.Background()

    // 10초 동안 메트릭 수집
    metrics, err := gcanalyzer.CollectForDuration(ctx, 10*time.Second, time.Second)
    if err != nil {
        panic(err)
    }
    
    // 수집된 데이터 분석
    analysis, err := gcanalyzer.Analyze(metrics)
    if err != nil {
        panic(err)
    }
    
    // 리포트 생성
    gcanalyzer.GenerateSummaryReport(analysis, os.Stdout)

    // 헬스 상태 확인
    health := gcanalyzer.GenerateHealthCheck(analysis)
    fmt.Printf("GC 헬스 점수: %d/100 (%s)\n", health.Score, health.Status)
}
```

### 알림과 함께 지속적 모니터링

```go
monitor := gcanalyzer.NewMonitor(&gcanalyzer.MonitorConfig{
        Interval:   time.Second,
    MaxSamples: 1000,
    OnAlert: func(alert *gcanalyzer.Alert) {
        log.Printf("[%s] %s: %s", alert.Severity, alert.Type, alert.Message)
    },
        OnMetric: func(m *gcanalyzer.GCMetrics) {
        log.Printf("Heap: %s, GC#: %d", 
            types.FormatBytes(m.HeapAlloc), m.NumGC)
    },
})

ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
defer cancel()

monitor.Start(ctx)
defer monitor.Stop()

// 애플리케이션 로직...
```

## API 참조

### 핵심 함수

| 함수 | 설명 |
|------|------|
| `CollectOnce()` | 단일 GC 메트릭 스냅샷 수집 |
| `CollectForDuration(ctx, duration, interval)` | 일정 기간 메트릭 수집 |
| `Analyze(metrics)` | 종합 GC 분석 수행 |
| `AnalyzeWithEvents(metrics, events)` | 상세 이벤트와 함께 분석 |
| `GenerateTextReport(analysis, w)` | 상세 텍스트 리포트 생성 |
| `GenerateJSONReport(analysis, w, indent)` | JSON 리포트 생성 |
| `GenerateSummaryReport(analysis, w)` | 간략한 요약 생성 |
| `GenerateHealthCheck(analysis)` | 헬스 체크 상태 생성 |

### 메트릭 타입

```go
// GCMetrics - 종합 GC 통계
type GCMetrics struct {
    NumGC         uint32    // 총 GC 사이클 수
    PauseTotalNs  uint64    // 총 pause 시간
    HeapAlloc     uint64    // 할당된 힙 바이트
    HeapSys       uint64    // 시스템 힙 바이트
    GCCPUFraction float64   // GC CPU 사용 비율
    // ... 추가 필드
}

// GCAnalysis - 분석 결과
type GCAnalysis struct {
    GCFrequency   float64       // 초당 GC 횟수
    AvgPauseTime  time.Duration // 평균 pause 시간
    P95PauseTime  time.Duration // 95번째 백분위 pause
    P99PauseTime  time.Duration // 99번째 백분위 pause
    AllocRate     float64       // 초당 할당 바이트
    GCOverhead    float64       // GC CPU 백분율
    Recommendations []string    // 최적화 제안
}
```

---

## 성능 벤치마킹 & 프로파일링 가이드

### 벤치마크 실행

```bash
# 모든 벤치마크 실행
make bench

# 빠른 벤치마크 (단일 실행)
make bench-short

# 비교를 위한 기준선 저장
make bench-save

# 현재 vs 기준선 비교
make bench-compare
```

### 수동 벤치마크 명령어

```bash
# 6회 반복 전체 벤치마크
go test -bench=. -benchmem -count=6 ./tests/... | tee bench.txt

# 두 벤치마크 파일 비교
benchstat baseline.txt current.txt
```

### CPU 프로파일링

```bash
# CPU 프로파일 생성
make bench-cpu

# 브라우저에서 보기
make pprof-cpu
# 또는 수동으로:
go tool pprof -http=:8080 profiles/cpu.prof
```

### 메모리 프로파일링

```bash
# 메모리 프로파일 생성
make bench-mem

# 브라우저에서 보기
make pprof-mem
# 또는 수동으로:
go tool pprof -http=:8080 profiles/mem.prof
```

### GC 추적

```bash
# GC 추적 활성화
GODEBUG=gctrace=1 go run ./examples/advanced/main.go

# 스케줄러 추적
GODEBUG=schedtrace=1000 go run ./examples/advanced/main.go

# GC pacer 추적
GODEBUG=gcpacertrace=1 go run ./examples/advanced/main.go
```

### gctrace 출력 이해하기

```
gc 1 @0.012s 2%: 0.015+0.89+0.003 ms clock, 0.12+0.45/0.67/0+0.024 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
```

| 필드 | 의미 |
|------|------|
| `gc 1` | GC 사이클 번호 |
| `@0.012s` | 프로그램 시작 이후 시간 |
| `2%` | GC에 소요된 CPU 시간 |
| `0.015+0.89+0.003 ms clock` | Wall clock: STW 마킹 + 동시 실행 + STW 스윕 |
| `4->4->0 MB` | 힙: 전 -> 후 -> 라이브 |
| `5 MB goal` | 목표 힙 크기 |
| `8 P` | 사용된 프로세서 수 |

---

## 성능 최적화 결과

### 벤치마크 비교 (최적화 전 → 후)

| 벤치마크 | 시간 | 메모리 | 할당 횟수 |
|----------|------|--------|-----------|
| GenerateTextReport | **-44%** | - | **-71%** (41→12) |
| MemoryUsage | **-20%** | - | **-67%** (45→15) |
| Analyzer_Analyze | **-5%** | **-55%** | **-33%** |
| AnalyzeSmallDataset | - | **-98%** | **-33%** |
| RealWorldScenario | - | -2% | **-17%** |
| **전체 (geomean)** | **-7.6%** | **-31%** | **-24%** |

### 적용된 주요 최적화

1. **재사용 가능한 슬라이스용 sync.Pool**
   - 분석기의 Duration 슬라이스
   - 리포터의 strings.Builder
   - GC 압력 크게 감소

2. **사전 할당 용량**
   - 권장사항 슬라이스: `make([]string, 0, 8)`
   - 헬스체크 이슈: `make([]string, 0, 6)`
   - 슬라이스 확장 재할당 방지

3. **fmt.Sprintf 대신 strings.Builder**
   - 텍스트 리포트 생성 최적화
   - 71% 적은 할당

4. **경량 메트릭 수집**
   - `NewGCMetricsLite()` - pause 데이터 생략 (~4KB 절약)
   - `NewGCMetricsPooled()` - pause 슬라이스 재사용

---

## 런타임 튜닝 가이드

### GOGC (GC 목표 백분율)

```bash
# 기본값은 100 (힙이 2배가 되면 GC)
GOGC=100 ./myapp

# 더 공격적인 GC (낮은 지연, 더 많은 CPU)
GOGC=50 ./myapp

# 덜 공격적인 GC (낮은 CPU, 더 많은 메모리)
GOGC=200 ./myapp
```

### GOMEMLIMIT (메모리 제한)

```bash
# 소프트 메모리 제한 설정 (Go 1.19+)
GOMEMLIMIT=1GiB ./myapp
```

### 사용 사례별 권장 설정

| 사용 사례 | GOGC | GOMEMLIMIT | 참고 |
|-----------|------|------------|------|
| 낮은 지연 | 50-100 | 자동 | 더 자주, 짧은 GC |
| 높은 처리량 | 200-400 | 설정 | GC 오버헤드 감소 |
| 메모리 제한 환경 | 50 | 제한 설정 | OOM 방지 |
| 배치 처리 | 400+ | 제한 설정 | GC 중단 최소화 |

---

## 프로젝트 구조

```
go-gc-analyzer/
├── pkg/
│   ├── gcanalyzer/    # 공개 API
│   │   └── api.go
│   └── types/         # 공유 타입
│       ├── metrics.go
│       ├── constants.go
│       ├── errors.go
│       └── format.go
├── internal/
│   ├── analysis/      # GC 분석 로직
│   ├── collector/     # 메트릭 수집
│   └── reporting/     # 리포트 생성
├── examples/
│   ├── basic/         # 간단한 사용 예제
│   ├── advanced/      # 고급 기능
│   └── monitoring/    # 지속적 모니터링
├── tests/
│   ├── analyzer_test.go
│   ├── benchmark_test.go
│   ├── collector_test.go
│   └── integration_test.go
├── benchmarks/        # 벤치마크 결과
├── profiles/          # 프로파일링 출력
├── Makefile           # 빌드 & 개발 명령어
└── README.md
```

---

## 예제

### 예제 실행

```bash
# 기본 예제
go run ./examples/basic/main.go

# 고급 예제
go run ./examples/advanced/main.go

# 모니터링 예제
go run ./examples/monitoring/main.go
```

---

## 개발

### 필수 요구사항

- Go 1.23+
- Make (선택, 편의 명령어용)

### 설정

```bash
# 저장소 클론
git clone https://github.com/kyungseok-lee/go-gc-analyzer.git
cd go-gc-analyzer

# 개발 도구 설치
make deps-tools

# 테스트 실행
make test

# 벤치마크 실행
make bench
```

### 사용 가능한 Make 타겟

```bash
make help        # 모든 명령어 표시
make test        # 테스트 실행
make bench       # 벤치마크 실행
make lint        # 린터 실행
make fmt         # 코드 포맷
make clean       # 아티팩트 정리
```

---

## 코드 품질

이 프로젝트는 모든 품질 검사를 통과합니다:

```bash
# 포맷 검사
gofmt -s -l .           # 출력 없음 = 모두 포맷됨

# 정적 분석
go vet ./...            # 모든 검사 통과

# 종합 린팅
golangci-lint run       # 모든 검사 통과
```

### 활성화된 린터

- **errcheck**: 에러 반환값 검사
- **staticcheck**: Go 정적 분석 (SA* 검사)
- **gosimple**: 코드 단순화 제안
- **govet**: 의심스러운 구조 보고
- **ineffassign**: 비효율적 할당 감지

---

## 기여하기

1. 저장소 Fork
2. 기능 브랜치 생성 (`git checkout -b feature/amazing`)
3. 테스트와 벤치마크 실행
4. 적절한 메시지 형식으로 커밋:
   - `perf:` 성능 개선
   - `feat:` 새 기능
   - `fix:` 버그 수정
   - `docs:` 문서화
   - `test:` 테스트 추가
5. 브랜치 푸시
6. 벤치마크 비교와 함께 Pull Request 생성

---

## 라이센스

MIT 라이센스 - LICENSE 파일 참조

---

## 참고 자료

- [Go GC 가이드](https://tip.golang.org/doc/gc-guide)
- [runtime 패키지 문서](https://pkg.go.dev/runtime)
- [pprof 문서](https://pkg.go.dev/runtime/pprof)
- [GODEBUG 환경 변수](https://pkg.go.dev/runtime#hdr-Environment_Variables)
