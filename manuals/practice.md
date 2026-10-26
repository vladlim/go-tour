# GO

---

**TASK**

Реализуйте функцию Chunk

```go
package main

import "fmt"

func Chunk(s []int, size int) [][]int {
	if size <= 0 {
		return [][]int{}
	}

	if size == len(s) {
		return [][]int{s}
	}

	res := make([][]int, 0, len(s)/size)

	for i := 0; i < len(s); i += size {
		boarder := min(i+size, len(s))
		res = append(res, s[i:boarder])
	}

	return res
}

func main() {
	fmt.Println(Chunk([]int{1, 2,3, 4}, 3))
}
```

Можно еще инициализировать каждый чанк через make([]int, boarder-i) + copy или append, но это лишние аллокации.

Но при этом если у нас изменится s, то чанки не поменяются.

generic-версия:

```go
func Chunk[T any](s []T, size int) [][]T {
	if size <= 0 {
		return [][]T{}
	}

	if size == len(s) {
		return [][]T{s}
	}

	res := make([][]T, 0, len(s)/size)

	for i := 0; i < len(s); i += size {
		boarder := min(i+size, len(s))
		res = append(res, s[i:boarder])
	}

	return res
}
```

---

**TASK**

Что будет выведено при запуске программы?

```go
package main

import (
  "fmt"
  "sync"
  "time"
)

func main() {
	counter := make([]int, 0, 1000)

	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			counter = append(counter, i)
			time.Sleep(10 * time.Second)
		}()
	}

	wg.Wait()

	fmt.Println(len(counter))
}
```

Примерно 1000

append конкурентно не безопасен

Лучше вынести в отдельную структуру, либо append обернуть в mutex.

---

**TASK**

Что выведет функция и почему?

```go
func getKeyAddress(m map[int]int, key int) *int {
  valAddr := &m[key]
  return valAddr
}

func main() {
  m := make(map[int]int)
  m[1] = 1
  fmt.Println(getKeyAddress(m, 1))
}
```

Программа не скомпилируется, потому что нельзя брать адрес у элемента мапы.

Обусловлено внутренним устройством мапы, она оптимально перестраивается сама при изменениях, поэтому адрес может начать указывать на мусор.

Если нам нужна именно ссылка на значение, а не на элемент мапы, можно поместить его в переменную и взять у нее адрес.

---

**TASK**

Написать реализацию функции по сжатию строки

```go
package main

import (
	"fmt"
	"strings"
)

func compactString(s string) string {
	var res strings.Builder

	b := []byte(s)

	i, j := 0, 0

	for j != len(s) {
		if s[i] == s[j] {
			j++
		} else {
			res.WriteByte(b[i])
			res.WriteByte(byte(j - i + 48))
			i = j
		}
	}

	res.WriteByte(b[i])
	res.WriteByte(byte(j - i + 48))

	return res.String()
}

func main() {
	fmt.Println(compactString("hhhhgggrreeee")) // h4g3r2e4
	fmt.Println(compactString("zzzzz"))         // z5
	fmt.Println(compactString("z"))             // z1
	fmt.Println(compactString("za"))            // z1a1
}
```

strings.Builder используется, потому что можно переиспользовать буффер и нет дополнительных выделений памяти при записи (только если буффер кончается - аллокация)

При кириллице нужно будет поменять на руны и конвертить cnt в руну.

Вообще скорее всего эталонное решение такое:

```go
func compactString(s string) string {
  var res strings.Builder

  cnt := 1

  for i := 0; i < len(s); i++ {
    if i < len(s) - 1 && s[i] == s[i+1] {
      cnt++
    } else {
      res.WriteString(fmt.Sprintf("%c%d", s[i], cnt))
    }
  }

  return res.String()
}
```

---

**TASK**

Написали hello world, собрали бинарь, почему бинарь весит 2мб,

Потому что в нем все равно содержатьтся fmt, библиотеки, которые по умолчанию нужны для работы программы, типа runtime.

---

**TASK**

Как работает планировщик в Go?

Модель PMG (Processor-Machine-Goroutine)

Goroutine - легковесный runtime поток в Go.

Machine thread - системный поток

Processor - логический процессор, к которому мапится системный поток и вместе P+M могут исполнять горутины.

К процессору привязываются очереди горутин.

По умолчанию количество P берется из значения GOMAXPROCS = кол-во логических ядер процессора.

Работает так:

Когда мы запускаем горутину, она встает в очередь горутин.

У горутин есть состояния (выполняется, ждет пока ее возьмет P+M, ждет мьютексы/канал, умерла)

То есть по идее единомомента выполняется не больше, чем P+M горутин.

При этом планировщик умеет вытеснять долго выполняющиеся горутины и ставить их на ождание.

Также планировщик может разгружать пары M+P, если видит, что у одной пары много очередей горутин, а у другой мало.

---

**TASK**

Как работает сборка мусора в Go

Для начала: есть стек и куча.

На стек кладутся в основном данные для быстрых вычислений, например, аргументы функций.

В кучу попадают долгоживущие объекты, которые могут понадобиться разным частям программы еще какое-то время.

Сборщик мусора работает параллельно с работой программы, занимается в основном чисткой кучи.

Есть основные две фазы: маркировка и чистка.

Так как полностью параллельно чистить мусор невозможно, есть Stop-The-World - механизм кратковременной остановки всех горутин.

Объект недостижим по ссылкам = мусор

Объект достижим по ссылкам = пока не чистим

Для разметки используется конкурентная трехцветная маркировка

GC начинает ходить от roots - корневых элементов типа глобальных переменных, стеков горутин, runtime структур и "красит" их в белый, серый и черный.

Цвета:
Белый - объект потенциально мусор
Серый - объект нужен, но еще не все его ссылки просмотрели
Черный - объект нужен

Сначала все объекты белые, потом при проходе их красят, пока не останется серых объектов.

Для завершения маркировки вызывается STW

Маркировка завершается

Затем вызывается STW еще раз

Чистка завершается.

Через GOGC можно настроить, когда GC должен будет срабатывать. В GOGC указывается на сколько процентов должна увеличиться куча после последней чистки, чтобы GC отработал.

Еще можно его руками вызвать через runtime.GC()

---

**TASK**

Есть набор урлов.

```go
package main

func main() {
	var urls = []string{
		"http://ozon.ru",
		"https://ozon.ru",
	}
}
```

Напишите программу, которая:

1. Поочередно выполнит http запросы по предложенному списку ссылок

- в случае получения http-кода ответа на запрос "200 OK" печатаем на экране "адрес url - ok"
- в случае получения http-кода ответа на запрос отличного от "200 OK" либо в случае ошибки печатаем на экране "адрес url - not ok"

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Result struct {
	URL string
	OK  bool
}

func main() {
	urls := []string{
		"http://mail.ru",
		"https://mail.ru",
		"http://google.com",
		"http://somesite.com",
		"http://non-existent.domain.tld",
		"https://ya.ru",
		"http://ya.ru",
		"http://eeee",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	jobs := make(chan string)
	results := make(chan Result)

	workersCount := 3

	var wg sync.WaitGroup

	for i := 0; i < workersCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for url := range jobs {
				ok := checkURL(client, url)

				results <- Result{
					URL: url,
					OK:  ok,
				}
			}
		}()
	}

	go func() {
		for _, url := range urls {
			jobs <- url
		}

		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.OK {
			fmt.Printf("address %s - ok\n", result.URL)
		} else {
			fmt.Printf("address %s - not ok\n", result.URL)
		}
	}
}

func checkURL(client *http.Client, url string) bool {
	resp, err := client.Get(url)
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode == http.StatusOK
}
```

---

**TASK**

Нужно написать простую библиотеку in-memory cache

Для простоты считаем, что у нас бесконечная память и нам не нужно задумываться об удалении ключей из него.

Реалзация должна удобвлетворять интерфейсу:

```go
type Cache interface {
	Set(k, v string)
	Get(k string) (v string, ok bool)
}
```

```go
package local_cache

import (
	"sync"
)

type Cache interface {
	Set(k, v string)
	Get(k string) (v string, ok bool)
}

type localCache struct {
	mu *sync.RWMutex
	data map[string]string
}

func New() *localCache {
	return &localCache{
		data: make(map[string]string)
	}
}

func (*localCache) Set(k, v string) {
	mu.Lock()
	defer mu.Unlock()

	data[k] = v
}

func (*localCache) Get(k string) (v string, ok bool) {
	mu.Rlock()
	defer mu.RUnlock()

	v, ok = data[k]
}

```

---

**TASK**

Что такое defer? Что выведет следующий код?

```go
type X struct {
	V int
}

func (x X) S() {
	fmt.Println(x.V)
}

func main() {
	x := X{123}
	defer x.S()
	x.V = 456
}
```

Выведет 123

Если поменять код на

```go
type X struct {
	V int
}

func (x X) S() {
	fmt.Println(x.V)
}

func main() {
	x := X{123}
	defer func() { x.S() } ()
	x.V = 456
}
```

Выведет 456

А если поставить несколько defer:

Они вызовутся в обратном порядке, потому что складываются в стек вызовов.

**TASK**

Можно ли передать переменную в несколько горутин? Пример (с ошибкой) - что будет?

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	m := make(map[int]int, 1)

	go func() { m[1] = 2 }()
	go func() { m[1] = 7 }()
	go func() { m[1] = 10 }()

	time.Sleep(3 * time.Second)

	fmt.Println(m[1])
}
```

Будет паника `concurrent map writes`.

Можно исправить через mutex:

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	m := make(map[int]int, 1)
	mu := sync.Mutex{}

	go func() {
		mu.Lock()
		m[1] = 2
		mu.Unlock()
	} ()

	go func() {
		mu.Lock()
		m[1] = 7
		mu.Unlock()
	} ()

	go func() {
		mu.Lock()
		m[1] = 10
		mu.Unlock()
	} ()

	time.Sleep(3 * time.Second)

	fmt.Println(m[1])
}
```

# Database

---

**TASK**

```sql
-- таблица клиентов
CREATE TABLE customers (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  city VARCHAR(255) NOT NULL
);

-- таблица товаров
CREATE TABLE products (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  category VARCHAR(100)
);

-- таблица позиций заказа
CREATE TABLE order_items (
  order_id BIGINT NOT NULL,
  product_id BIGINT NOT NULL,
  qty DECIMAL(10, 3) NOT NULL CHECK (qty > 0), -- 10 цифр, 3 знака после запятой
  price NUMERIC(10, 2) NOT NULL CHECK (price >= 0)
);

-- таблица заказа
CREATE TABLE orders (
  id BIGSERIAL PRIMARY KEY,
  customer_id BIGINT,
  order_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'canceled'))
);

```

1. Выведите топ 10 по сумме заказов пользователей, совершивших более 2-х заказов за последние 30 дней.

```sql
SELECT
  c.id AS customer_id,
  c.name AS customer_name,
  c.city AS customer_city,
  COUNT(DISTINCT o.id) AS order_count,
  SUM(oi.price * oi.qty) AS total_price
FROM customers c
JOIN order o ON c.id = o.customer_id
JOIN order_items oi ON o.id = oi.order_id
WHERE o.status = 'paid' AND o.order_date > NOW() - INTERVAL '30 days'
GROUP BY c.id, c.name, c.city
HAVING COUNT(DISTINCT o.id) > 2
ORDER BY total_price DESC
LIMIT 10;
```

2. Выведите топ 3 категорий по выручке за последние 6 месяцев

```sql
SELECT
  p.category as category,
  SUM(oi.price * oi.qty) AS total_price
FROM products p
JOIN order_items oi ON p.id = oi.product_id
JOIN orders o ON oi.order_id = o.id
WHERE o.status = 'paid'
  AND o.order_date > NOW() - INTERVAL '6 months'
GROUP BY p.category
ORDER BY total_price DESC
LIMIT 3;
```

---

**TASK**

```sql
-- таблица клиентов
CREATE TABLE customers (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  city VARCHAR(255) NOT NULL
);

-- таблица товаров
CREATE TABLE products (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  category VARCHAR(100)
);

-- таблица позиций заказа
CREATE TABLE order_items (
  order_id BIGINT NOT NULL,
  product_id BIGINT NOT NULL,
  qty DECIMAL(10, 3) NOT NULL CHECK (qty > 0), -- 10 цифр, 3 знака после запятой
  price NUMERIC(10, 2) NOT NULL CHECK (price >= 0)
);

-- таблица заказа
CREATE TABLE orders (
  id BIGSERIAL PRIMARY KEY,
  customer_id BIGINT,
  order_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'canceled'))
);

```

1. Выведите топ-3 города по выручке в убывающем порядке за последние 30 дней

```sql
SELECT
  c.city AS city,
  SUM(oi.price * oi.qty) AS total_price
FROM customers c
JOIN orders o ON c.id = o.customer_id
JOIN order_items oi ON o.id = oi.order_id
WHERE o.status = 'paid' AND o.order_date > NOW() - INTERVAL '30 days'
GROUP BY c.city
ORDER BY total_price DESC
LIMIT 3;
```

2. Выведите все товары у которых не было продаж за последние полгода

```sql
SELECT
  p.id AS product_id,
  p.name AS product_name,
  p.category AS category
FROM products p
LEFT JOIN order_items oi ON oi.product_id = p.id
LEFT JOIN orders o
  ON o.id = oi.order_id
  AND o.order_date >= NOW() - INTERVAL '6 month'
  AND o.status = 'paid'
GROUP BY p.id, p.name, p.category
HAVING COUNT(o.id) = 0;
```

---

**TASK**

Построить оптимальный индекс для

```sql
SELECT * FROM employee
WHERE sex = 'm' AND salary > 300000 AND age = 20
ORDER BY created_at;
```

Нужно смотреть на селективность по условиям (то есть какой процент строк от всей таблицы возвращается по конкретному условию)

Нужно сначала посмотреть на условия по равенству и расположить их в порядке ухудшения селективности.

Если у нас sex='m' имеет плохую селективность, то нет смысла делать по нему индекс.

Лучше тогда сделать индекс (age, salary, created_at)

---

# Architecture
