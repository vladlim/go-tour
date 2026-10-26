# Практикум по профилированию Go

Проект специально содержит две реализации одной CPU-задачи и две реализации
аллокационной задачи. Они дают заметную разницу в benchmark и профилях.

Нужен Go 1.22+. Все команды выполняются из этой директории.

## 1. Проверка и benchmark

```bash
go test ./...
go test -run='^$' -bench=. -benchmem -count=5
```

Сравните `ns/op`, `B/op` и `allocs/op` у sub-benchmark `slow/fast` и
`wasteful/reuse`. Числа конкретной машины не важны — важны порядок величин и
стабильность нескольких запусков.

## 2. Профиль из benchmark

Профилируем только медленную реализацию, чтобы результаты не смешались:

```bash
mkdir -p profiles
go test -run='^$' -bench='BenchmarkPrimes/slow$' -benchtime=5s \
  -cpuprofile=profiles/bench-slow-cpu.out
go tool pprof ./profile.test profiles/bench-slow-cpu.out
```

В интерактивном prompt выполните:

```text
top
top -cum
list countPrimesSlow
peek countPrimesSlow
```

Для web UI:

```bash
go tool pprof -http=:0 ./profile.test profiles/bench-slow-cpu.out
```

Профиль аллокаций:

```bash
go test -run='^$' -bench='BenchmarkAllocations/wasteful$' -benchtime=3s \
  -memprofile=profiles/bench-alloc.out
go tool pprof -sample_index=alloc_space ./profile.test profiles/bench-alloc.out
```

Внутри попробуйте `top` и `list allocateWastefully`. `alloc_space` выбран потому,
что буферы быстро становятся мусором; в live heap они могут уже отсутствовать.

## 3. CLI без HTTP: `runtime/pprof`

```bash
go build -o profile-demo .
./profile-demo -task=cpu -impl=slow -n=100000 -repeat=100 \
  -cpuprofile=profiles/cli-slow.out
go tool pprof ./profile-demo profiles/cli-slow.out
```

Если запуск слишком короткий и samples мало, увеличьте `-repeat`. Heap-профиль:

```bash
./profile-demo -task=alloc -impl=slow -n=10000 -repeat=20 \
  -memprofile=profiles/cli-heap.out
go tool pprof ./profile-demo profiles/cli-heap.out
```

Пример перед heap-снимком вызывает GC, поэтому он показывает удерживаемую память,
а временные буферы уже могут исчезнуть. Для общего потока аллокаций лучше подходит
benchmark-профиль с `alloc_space` выше.

## 4. Живой HTTP-процесс: `net/http/pprof`

Терминал 1:

```bash
./profile-demo -mode=server -task=cpu -impl=slow -n=100000 -repeat=20
```

Терминал 2 — создаём нагрузку в фоне и одновременно пишем CPU-профиль:

```bash
while true; do curl -s http://localhost:6060/work >/dev/null; done
```

Терминал 3:

```bash
go tool pprof -seconds=20 ./profile-demo \
  http://localhost:6060/debug/pprof/profile
```

Остановите цикл нагрузки через `Ctrl+C`. Моментальные профили и goroutine dump:

```bash
go tool pprof ./profile-demo http://localhost:6060/debug/pprof/heap
curl 'http://localhost:6060/debug/pprof/goroutine?debug=2'
```

## 5. Сравнение slow и fast

Снимки должны иметь одинаковые параметры нагрузки. Быстрая реализация может
закончить работу раньше, поэтому подберите `-repeat`, чтобы оба запуска длились
несколько секунд:

```bash
./profile-demo -impl=slow -repeat=200 -cpuprofile=profiles/before.out
./profile-demo -impl=fast -repeat=200 -cpuprofile=profiles/after.out
go tool pprof -diff_base profiles/before.out ./profile-demo profiles/after.out
```

В prompt выполните `top`. Отрицательные значения означают уменьшение относительно
baseline. Но из-за разной длительности профилей процентное сравнение может быть
неинтуитивным, поэтому подтвердите результат benchmark-серией:

```bash
go test -run='^$' -bench=BenchmarkPrimes -benchmem -count=10 > profiles/bench.txt
```

Для строгого сравнения двух версий кода сохраните вывод до и после отдельно и
используйте `benchstat before.txt after.txt`.

## 6. Самостоятельные эксперименты

1. Уберите `runtime.GC()` перед heap-профилем и объясните разницу.
2. Добавьте утечку goroutine, ожидающих никогда не закрываемый channel, и найдите
   одинаковые стеки в goroutine profile.
3. Добавьте общий mutex и параллельные запросы; включите
   `runtime.SetMutexProfileFraction(1)` и изучите `/debug/pprof/mutex`.
4. Запустите `go test -race ./...`: race detector и profiler решают разные задачи.
5. Снимите `go tool trace` для benchmark (`go test -trace=trace.out ...`) и сравните
   временную шкалу trace с агрегированным представлением pprof.

Папка `profiles/` и бинарник `profile-demo` — локальные артефакты; они исключены
из Git через `.gitignore`.
