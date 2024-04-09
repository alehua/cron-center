package main

import (
	"context"
	"github.com/alehua/cron-center/internal/boost"
	"log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := boost.WithStart(ctx); err != nil {
		log.Panicln(err)
	}
}
