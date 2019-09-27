package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
)

func TestMain(m *testing.M) {
	fmt.Println("Wait 5s for service availability...")
	time.Sleep(5 * time.Second)

	status := godog.RunWithOptions("integration", func(s *godog.Suite) {
		FeatureContext(s)
	}, godog.Options{
		Format:    "progress", // Замените на "pretty" для лучшего вывода
		Paths:     []string{"features"},
		Randomize: 0, // Последовательный порядок исполнения
	})

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}
