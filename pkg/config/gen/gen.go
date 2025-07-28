package main

import (
	cfg "github.com/conductorone/baton-rippling/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("rippling", cfg.Config)
}
