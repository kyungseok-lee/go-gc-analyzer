# Go GC 분석기

[![Go Report Card](https://goreportcard.com/badge/github.com/kyungseok-lee/go-gc-analyzer)](https://goreportcard.com/report/github.com/kyungseok-lee/go-gc-analyzer)
[![GoDoc](https://godoc.org/github.com/kyungseok-lee/go-gc-analyzer?status.svg)](https://godoc.org/github.com/kyungseok-lee/go-gc-analyzer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Go 애플리케이션의 가비지 컬렉션(GC) 성능을 분석하고 모니터링하는 포괄적인 Go 라이브러리입니다. 이 라이브러리는 GC 동작, 메모리 사용 패턴, 성능 메트릭에 대한 상세한 인사이트를 제공하여 Go 애플리케이션 최적화를 돕습니다.

## 🚀 주요 기능

- **실시간 GC 모니터링**: 설정 가능한 간격으로 GC 메트릭을 지속적으로 수집
- **포괄적인 분석**: GC 빈도, 일시 정지 시간, 메모리 사용량, 할당 패턴의 상세 분석
- **다양한 리포트 형식**: 텍스트, JSON, 테이블, Prometheus 형식의 리포트 생성
- **헬스 모니터링**: 설정 가능한 알림 임계값을 가진 내장 헬스 체크
- **메모리 트렌드 분석**: 시간에 따른 메모리 사용 패턴 추적
- **일시 정지 시간 분포**: GC 일시 정지 시간 분포 및 백분위수 분석
- **성능 권장사항**: GC 성능 최적화를 위한 자동화된 제안
- **HTTP 엔드포인트**: 메트릭 노출을 위한 바로 사용 가능한 HTTP 서버
- **의존성 없음**: 외부 의존성이 없는 순수 Go 구현

## 📦 설치

```bash
go get github.com/kyungseok-lee/go-gc-analyzer
```

## 🏃‍♂️ 빠른 시작

### 기본 사용법

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/kyungseok-lee/go-gc-analyzer/analyzer"
)

func main() {
    // 10초간 GC 메트릭 수집
    ctx := context.Background()
    metrics, err := analyzer.CollectForDuration(ctx, 10*time.Second, time.Second)
    if err != nil {
        panic(err)
    }
    
    // 수집된 메트릭 분석
    gcAnalyzer := analyzer.NewAnalyzer(metrics)
    analysis, err := gcAnalyzer.Analyze()
    if err != nil {
        panic(err)
    }
    
    // 분석 결과 출력
    fmt.Printf("GC 빈도: %.2f GCs/초\n", analysis.GCFrequency)
    fmt.Printf("평균 일시 정지 시간: %v\n", analysis.AvgPauseTime)
    fmt.Printf("평균 힙 크기: %s\n", formatBytes(analysis.AvgHeapSize))
    fmt.Printf("GC 오버헤드: %.2f%%\n", analysis.GCOverhead)
    
    // 리포트 생성
    reporter := analyzer.NewReporter(analysis, metrics, nil)
    reporter.GenerateSummaryReport(os.Stdout)
}
```

### 지속적인 모니터링

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/kyungseok-lee/go-gc-analyzer/analyzer"
)

func main() {
    config := &analyzer.CollectorConfig{
        Interval:   time.Second,
        MaxSamples: 300, // 5분간의 데이터 보관
        OnMetricCollected: func(m *analyzer.GCMetrics) {
            if m.GCCPUFraction > 0.1 {
                log.Printf("높은 GC CPU 사용률: %.2f%%", m.GCCPUFraction*100)
            }
        },
        OnGCEvent: func(e *analyzer.GCEvent) {
            if e.Duration > 10*time.Millisecond {
                log.Printf("긴 GC 일시 정지: %v", e.Duration)
            }
        },
    }
    
    collector := analyzer.NewCollector(config)
    
    ctx := context.Background()
    err := collector.Start(ctx)
    if err != nil {
        panic(err)
    }
    
    // 1분간 실행
    time.Sleep(1 * time.Minute)
    
    collector.Stop()
    
    // 수집된 데이터 분석
    metrics := collector.GetMetrics()
    if len(metrics) >= 2 {
        gcAnalyzer := analyzer.NewAnalyzer(metrics)
        analysis, _ := gcAnalyzer.Analyze()
        
        fmt.Printf("분석 완료: %d개의 권장사항\n", len(analysis.Recommendations))
        for _, rec := range analysis.Recommendations {
            fmt.Printf("- %s\n", rec)
        }
    }
}
```

## 📊 모니터링 서버

라이브러리에는 바로 사용 가능한 HTTP 모니터링 서버가 포함되어 있습니다:

```bash
go run examples/monitoring/main.go
```

다음 엔드포인트를 제공하는 모니터링 서비스를 시작합니다:

- `http://localhost:8080/metrics` - 현재 GC 메트릭 (JSON)
- `http://localhost:8080/health` - 헬스 체크 상태
- `http://localhost:8080/analysis` - 전체 GC 분석
- `http://localhost:8080/prometheus` - Prometheus 형식 메트릭
- `http://localhost:8080/trend` - 메모리 사용 트렌드
- `http://localhost:8080/distribution` - 일시 정지 시간 분포

## 📖 API 문서

### 핵심 타입

#### GCMetrics
특정 시점의 GC 메트릭 스냅샷을 나타냅니다.

```go
type GCMetrics struct {
    NumGC          uint32        // GC 횟수
    PauseTotalNs   uint64        // 총 일시 정지 시간(나노초)
    HeapAlloc      uint64        // 현재 힙 할당량
    TotalAlloc     uint64        // 총 할당된 바이트
    Sys            uint64        // OS로부터 받은 총 바이트
    GCCPUFraction  float64       // GC에 소요된 CPU 시간 비율
    Timestamp      time.Time     // 수집 시간
    // ... 더 많은 필드
}
```

#### GCAnalysis
포괄적인 분석 결과를 포함합니다.

```go
type GCAnalysis struct {
    Period           time.Duration  // 분석 기간
    GCFrequency      float64        // 초당 GC 횟수
    AvgPauseTime     time.Duration  // 평균 일시 정지 시간
    P95PauseTime     time.Duration  // 95번째 백분위수 일시 정지 시간
    P99PauseTime     time.Duration  // 99번째 백분위수 일시 정지 시간
    AvgHeapSize      uint64         // 평균 힙 크기
    AllocRate        float64        // 할당 속도 (바이트/초)
    GCOverhead       float64        // GC CPU 오버헤드 비율
    MemoryEfficiency float64        // 메모리 효율성 비율
    Recommendations  []string       // 성능 권장사항
    // ... 더 많은 필드
}
```

### 주요 함수들

#### 수집 함수

```go
// 단일 스냅샷 수집
func CollectOnce() *GCMetrics

// 특정 기간 동안 수집
func CollectForDuration(ctx context.Context, duration, interval time.Duration) ([]*GCMetrics, error)
```

#### 분석 함수

```go
// 메트릭으로부터 분석기 생성
func NewAnalyzer(metrics []*GCMetrics) *Analyzer

// 분석 수행
func (a *Analyzer) Analyze() (*GCAnalysis, error)

// 메모리 트렌드 데이터 가져오기
func (a *Analyzer) GetMemoryTrend() []MemoryPoint

// 일시 정지 시간 분포 가져오기
func (a *Analyzer) GetPauseTimeDistribution() map[string]int
```

#### 리포팅 함수

```go
// 리포터 생성
func NewReporter(analysis *GCAnalysis, metrics []*GCMetrics, events []*GCEvent) *Reporter

// 다양한 리포트 형식 생성
func (r *Reporter) GenerateTextReport(w io.Writer) error
func (r *Reporter) GenerateJSONReport(w io.Writer, indent bool) error
func (r *Reporter) GenerateTableReport(w io.Writer) error
func (r *Reporter) GenerateSummaryReport(w io.Writer) error
func (r *Reporter) GenerateGrafanaMetrics(w io.Writer) error

// 헬스 체크 생성
func (r *Reporter) GenerateHealthCheck() *HealthCheckStatus
```

## 🔧 설정

### 컬렉터 설정

```go
type CollectorConfig struct {
    // 수집 간격 (기본값: 1초)
    Interval time.Duration
    
    // 메모리에 보관할 최대 샘플 수 (기본값: 1000)
    MaxSamples int
    
    // 각 메트릭 수집 시 콜백
    OnMetricCollected func(*GCMetrics)
    
    // 각 GC 이벤트 시 콜백
    OnGCEvent func(*GCEvent)
}
```

### 알림 임계값

```go
type AlertThresholds struct {
    MaxGCFrequency   float64       // 초당 최대 GC 횟수
    MaxPauseTime     time.Duration // 최대 일시 정지 시간
    MaxGCOverhead    float64       // 최대 GC CPU 비율
    MinHealthScore   int           // 최소 헬스 점수
}
```

## 📈 메트릭 이해하기

### GC 빈도
- **낮음 (< 1 GC/초)**: 우수, 최소한의 GC 압박
- **보통 (1-5 GC/초)**: 양호, 정상적인 애플리케이션 동작
- **높음 (> 5 GC/초)**: 최적화 고려 필요, 할당 속도 감소 필요

### 일시 정지 시간
- **우수 (< 1ms)**: 저지연 애플리케이션
- **양호 (1-10ms)**: 대부분의 애플리케이션
- **주의 필요 (> 10ms)**: 응답성에 영향을 줄 수 있음
- **심각 (> 100ms)**: 즉시 최적화 필요

### GC 오버헤드
- **우수 (< 5%)**: 최소한의 GC 영향
- **양호 (5-15%)**: 대부분의 애플리케이션에 허용 가능
- **높음 (15-25%)**: 튜닝 고려 필요
- **심각 (> 25%)**: 상당한 성능 영향

### 메모리 효율성
- **우수 (> 80%)**: 효율적인 메모리 사용
- **양호 (60-80%)**: 정상적인 사용
- **나쁨 (< 60%)**: 메모리 단편화 또는 비효율적인 할당 패턴

## 🎯 성능 최적화 팁

분석 결과를 바탕으로 한 일반적인 최적화 전략:

### 높은 GC 빈도
- 객체 재사용으로 할당 속도 감소
- 자주 할당되는 객체에 대해 객체 풀 사용
- `GOGC` 값을 증가시켜 GC 발생 빈도 감소
- 포인터 간접 참조를 줄이도록 데이터 구조 최적화

### 긴 일시 정지 시간
- 가능하면 힙 크기 감소
- 대용량 객체 할당 최소화
- 배치 처리 대신 스트리밍 처리 사용
- 동시성 GC 튜닝 고려 (Go 1.19+)

### 높은 GC 오버헤드
- `go tool pprof`로 할당 핫스팟 프로파일링
- 객체 풀링 구현
- 가능한 곳에서 포인터 타입 대신 값 타입 사용
- 슬라이스와 맵 사용 패턴 최적화

### 메모리 누수
- 고루틴 누수 확인
- 리소스의 적절한 정리 보장
- 적절한 곳에 약한 참조 사용
- 시간에 따른 메모리 증가 트렌드 모니터링

## 🏷️ 예시

라이브러리에는 포괄적인 예시가 포함되어 있습니다:

- **[기본 사용법](examples/basic/main.go)**: 간단한 수집과 분석
- **[고급 기능](examples/advanced/main.go)**: 워크로드 분석, 성능 비교
- **[모니터링 서비스](examples/monitoring/main.go)**: 알림이 있는 HTTP 모니터링 서버

예시 실행:

```bash
# 기본 예시
go run examples/basic/main.go

# 고급 기능
go run examples/advanced/main.go

# 모니터링 서비스
go run examples/monitoring/main.go
```

## 🧪 테스트

전체 테스트 스위트 실행:

```bash
# 모든 테스트 실행
go test ./...

# 상세 출력으로 실행
go test -v ./...

# 벤치마크 실행
go test -bench=. ./tests

# 레이스 검출과 함께 실행
go test -race ./...

# 커버리지 리포트 생성
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📊 벤치마크

라이브러리는 최소한의 오버헤드를 위해 설계되었습니다:

```
BenchmarkCollectOnce-8                    100000    10235 ns/op    2048 B/op     12 allocs/op
BenchmarkAnalyzer_Analyze-8                5000      234567 ns/op   45678 B/op   123 allocs/op
BenchmarkReporter_GenerateTextReport-8    10000     102345 ns/op   12345 B/op    45 allocs/op
```

성능 특성:
- **CollectOnce**: 수집당 약 10μs
- **분석**: 데이터 포인트와 선형적으로 확장
- **리포팅**: 모든 형식의 빠른 생성
- **메모리 오버헤드**: 최소한, 설정 가능한 보관 기간

## 🔌 통합

### Prometheus/Grafana

Prometheus 형식으로 메트릭 내보내기:

```go
reporter := analyzer.NewReporter(analysis, metrics, nil)
err := reporter.GenerateGrafanaMetrics(w)
```

### JSON API

모든 데이터 구조는 쉬운 통합을 위해 JSON 직렬화 가능:

```go
analysis, _ := gcAnalyzer.Analyze()
data, _ := json.Marshal(analysis)
```

### 헬스 체크

헬스 체크 시스템과 통합:

```go
healthCheck := reporter.GenerateHealthCheck()
if healthCheck.Status != "healthy" {
    // 알림 또는 조치 취하기
}
```

## 🤝 기여하기

기여를 환영합니다! Pull Request를 제출하시거나, 주요 변경사항의 경우 먼저 이슈를 열어 논의해 주세요.

### 개발 환경 설정

1. 저장소 포크
2. 기능 브랜치 생성 (`git checkout -b feature/amazing-feature`)
3. 변경사항 작성
4. 변경사항에 대한 테스트 추가
5. 테스트 스위트 실행 (`go test ./...`)
6. 변경사항 커밋 (`git commit -am 'Add amazing feature'`)
7. 브랜치에 푸시 (`git push origin feature/amazing-feature`)
8. Pull Request 열기

### 가이드라인

- 명확하고 자체 문서화된 코드 작성
- 새로운 기능에 대한 테스트 추가
- 필요에 따라 문서 업데이트
- Go 모범 사례와 관용구 따르기
- 가능한 경우 하위 호환성 보장

## 📄 라이선스

이 프로젝트는 MIT 라이선스 하에 라이선스됩니다 - 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 🙏 감사의 말

- 훌륭한 런타임 메트릭 API를 제공한 Go 팀
- 영감과 피드백을 준 Go 커뮤니티
- 이 라이브러리 개선에 도움을 주는 기여자들

## 📞 지원

- 📖 [문서](https://godoc.org/github.com/kyungseok-lee/go-gc-analyzer)
- 🐛 [이슈 트래커](https://github.com/kyungseok-lee/go-gc-analyzer/issues)
- 💬 [토론](https://github.com/kyungseok-lee/go-gc-analyzer/discussions)

---

**Go 커뮤니티를 위해 ❤️로 만들어졌습니다**