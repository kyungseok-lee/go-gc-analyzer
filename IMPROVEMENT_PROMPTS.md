# Go GC Analyzer 개선 프롬프트

## 📊 현재 프로젝트 분석 결과

| 항목 | 현재 상태 | 개선 필요 |
|------|----------|-----------|
| **코드 라인** | ~4,281 lines | - |
| **테스트 커버리지** | 0% (테스트가 tests/ 패키지에만 존재) | ⚠️ 높음 |
| **Go 버전** | 1.23 / 1.25.5 | ✅ 최신 |
| **외부 의존성** | 0개 (순수 표준 라이브러리) | ✅ 우수 |
| **린트/포맷** | 통과 | ✅ |
| **Makefile** | 완비 | ✅ |
| **문서화** | README 완비 | ✅ |

---

## 🎯 우선순위 및 작업 목록

| 순위 | 작업 | 상태 | 예상 시간 |
|------|------|------|-----------|
| 1 | 테스트 커버리지 개선 | ✅ 완료 | 4-6시간 |
| 2 | GitHub Actions CI 워크플로우 | ✅ 완료 | 1-2시간 |
| 3 | golangci-lint 설정 파일 | ✅ 완료 | 30분 |
| 4 | CHANGELOG.md 생성 | ✅ 완료 | 30분 |
| 5 | go.mod toolchain 정리 | ✅ 완료 | 10분 |

### 📈 달성된 테스트 커버리지
| 패키지 | 커버리지 |
|--------|----------|
| internal/analysis | 90.1% |
| internal/collector | 68.8% |
| internal/reporting | 96.6% |
| pkg/types | 98.2% |

---

## 1️⃣ 테스트 커버리지 개선

### 목표
- 각 패키지에 단위 테스트 추가
- 커버리지 목표: 60% 이상

### 작업 내용
1. `internal/analysis/analyzer_test.go` 생성
2. `internal/collector/collector_test.go` 생성
3. `internal/reporting/reporter_test.go` 생성
4. `pkg/types/metrics_test.go` 생성

### 테스트 작성 원칙
- Table-Driven Tests 패턴 사용
- 엣지 케이스 커버 (nil, 빈 슬라이스, 경계값)
- 동시성 테스트 (race condition)

### 테스트 케이스
```go
// Analyzer 테스트 케이스
- Analyze(): 정상 데이터, 데이터 부족, 경계값
- analyzeGCFrequency(): GC 없음, GC 많음
- analyzePauseTimes(): 이벤트 있음/없음
- GetMemoryTrend(): 빈 메트릭, 단일, 다수
- GetPauseTimeDistribution(): 각 버킷 테스트

// Collector 테스트 케이스
- Start/Stop: 정상, 중복 시작, 중복 정지
- IsRunning: 상태 확인
- GetMetrics/GetEvents: 빈 상태, 데이터 있음
- 동시성: race condition 테스트

// Reporter 테스트 케이스
- GenerateTextReport(): 정상, nil analysis
- GenerateJSONReport(): indent 옵션
- GenerateHealthCheck(): 각 상태별 테스트
- GenerateTableReport(): 빈 메트릭

// Types 테스트 케이스
- NewGCMetrics(): 정상 생성
- NewGCMetricsPooled(): 풀링 동작
- Release(): 풀 반환
- Clone(): 깊은 복사 확인
- FormatBytes(): 다양한 크기
```

---

## 2️⃣ GitHub Actions CI 워크플로우

### 목표
- PR/Push 시 자동 테스트, 린트, 빌드

### 파일 위치
`.github/workflows/ci.yml`

### 워크플로우 내용
```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: Test
        run: go test -v -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v4

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: golangci/golangci-lint-action@v6

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: Build
        run: go build ./...
```

---

## 3️⃣ golangci-lint 설정 파일

### 파일 위치
`.golangci.yml`

### 설정 내용
```yaml
run:
  timeout: 5m
  go: '1.23'

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - unconvert
    - gocritic
    - revive

linters-settings:
  errcheck:
    check-type-assertions: true
  govet:
    enable-all: true
  revive:
    severity: warning

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

---

## 4️⃣ CHANGELOG.md 생성

### 형식
Keep a Changelog 형식 (https://keepachangelog.com)

### 내용
```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.1.0] - 2026-01-06

### Added
- Initial release
- GC metrics collection (NewGCMetrics, NewGCMetricsPooled, NewGCMetricsLite)
- GC analysis with pause time percentiles (P95/P99)
- Multiple report formats (Text, JSON, Summary, Table, Prometheus)
- Health check scoring system
- Memory trend analysis
- Real-time monitoring with alerts
- sync.Pool optimizations for reduced allocations

### Performance
- 44% faster text report generation
- 55% reduction in memory allocations for analysis
- 71% fewer allocations in report generation
```

---

## 5️⃣ go.mod toolchain 정리

### 현재 상태
```go
module github.com/kyungseok-lee/go-gc-analyzer

go 1.23.0

toolchain go1.25.5
```

### 목표 상태
```go
module github.com/kyungseok-lee/go-gc-analyzer

go 1.23
```

### 이유
- toolchain 지시어는 Go 1.21+에서 자동 관리됨
- 명시적 버전만 지정하는 것이 깔끔함

---

## 📝 작업 완료 체크리스트

- [x] 1. 테스트 커버리지 개선
  - [x] internal/analysis/analyzer_test.go
  - [x] internal/collector/collector_test.go
  - [x] internal/reporting/reporter_test.go
  - [x] pkg/types/metrics_test.go
  - [x] 커버리지 60% 달성 (평균 88.4%)
- [x] 2. GitHub Actions CI
  - [x] .github/workflows/ci.yml 생성
  - [ ] 워크플로우 테스트 (푸시 후 확인)
- [x] 3. golangci-lint 설정
  - [x] .golangci.yml 생성
  - [x] 린트 통과 확인 (go vet)
- [x] 4. CHANGELOG.md 생성
- [x] 5. go.mod 정리
- [x] 6. Git commit & push

---

## 🚀 실행 명령

```bash
# 테스트 실행
go test -v -race -cover ./...

# 커버리지 리포트
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 린트 실행
golangci-lint run

# 벤치마크
make bench
```

