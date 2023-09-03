// v0.4
package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Кольцевой буфер.
type RingIntBuffer struct {
	array []int      // Более низкоуровневое хранилище нашего буфера.
	pos   int        // Текущая позиция кольцевого буфера.
	size  int        // Общий размер буфера.
	m     sync.Mutex // Мьютекс для потокобезопасного доступа к буферу. Так как одновременно могут быть вызваны
	//методы Get и Push. Первый - это когда настало время вывести содержимое буфера и очистить его.
	// Второй - это когда пользователь ввёл новое число. Оба события обрабатываются разными
	//горутинами.
}

// Создание кольцевого буфера.
func NewRingIntBuffer(size int) *RingIntBuffer {
	return &RingIntBuffer{make([]int, size), -1, size, sync.Mutex{}}
}

// Push добавление нового элемента в конец буфера.
// При попытке добавления нового элемента в заполненный буфер самое старое значение затирается.
func (r *RingIntBuffer) Push(el int) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.pos == r.size-1 {
		// Сдвигаем все элементы буфера на одну позицию в сторону начала.
		for i := 1; i <= r.size-1; i++ {
			r.array[i-1] = r.array[i]
		}
		r.array[r.pos] = el
	} else {
		r.pos++
		r.array[r.pos] = el
	}
}

// Get - получение всех элементов буфера и его последующая очистка.
func (r *RingIntBuffer) Get() []int {
	if r.pos <= 0 {
		return nil
	}
	r.m.Lock()
	defer r.m.Unlock()
	var output []int = r.array[:r.pos+1]
	// Виртуальная очистка нашего буфера
	r.pos = -1
	return output
}

func main() {
	input := make(chan int)
	done := make(chan bool)
	go read(input, done)

	// Фильтр негативных значений.
	negativeFilterChannel := make(chan int)
	go negativeFilterStageInt(input, negativeFilterChannel, done)

	// Фильтр значений кратных 3.
	notDivideThreeChannel := make(chan int)
	go notDivideThreeFunc(negativeFilterChannel, notDivideThreeChannel, done)

	// Буферизация канала
	bufferedIntChannel := make(chan int)
	bufferSize := 10
	bufferDrainInter := 30 * time.Second
	go bufferStageFunc(notDivideThreeChannel, bufferedIntChannel, done, bufferSize, bufferDrainInter)

	for {
		select {
		case data := <-bufferedIntChannel:
			log.Println("Обработанные данные, ", data)
		case <-done:
			return
		}
	}
}

// Реализация консоли ввода значений.
func read(nextStage chan<- int, done chan bool) {
	scanner := bufio.NewScanner(os.Stdin)
	var data string
	for scanner.Scan() {
		data = scanner.Text()
		if strings.EqualFold(data, "Exit") {
			log.Println("Программа завершила работу!")
			close(done)
			return
		}
		i, err := strconv.Atoi(data)
		if err != nil {
			log.Println("Программа обрабатывает только целые числа")
			continue
		}
		nextStage <- i
	}
}

// Реализация фильтра негативных значений
func negativeFilterStageInt(previousStageChannel <-chan int, nextStageChannel chan<- int, done <-chan bool) {
	for {
		select {
		case data := <-previousStageChannel:
			if data > 0 {
				nextStageChannel <- data
			}
		case <-done:
			return
		}
	}
}

// Реализация фильтра чисел кратных 3.
func notDivideThreeFunc(previousStageChannel <-chan int, nextStageChannel chan<- int, done <-chan bool) {
	for {
		select {
		case data := <-previousStageChannel:
			if data%3 == 0 {
				nextStageChannel <- data
			}
		case <-done:
			return
		}
	}
}

// Реализация буфера
func bufferStageFunc(previousStageChannel <-chan int, nextStageChannel chan<- int, done <-chan bool, size int, interval time.Duration) {
	buffer := NewRingIntBuffer(size)
	for {
		select {
		case data := <-previousStageChannel:
			buffer.Push(data)
		case <-time.After(interval):
			bufferData := buffer.Get()
			if bufferData != nil {
				for _, data := range bufferData {
					nextStageChannel <- data
				}
			}
		case <-done:
			return
		}
	}
}

