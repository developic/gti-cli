package assets

import "embed"

//go:embed words/*
var Words embed.FS

//go:embed themes/*
var Themes embed.FS

//go:embed code/*
var Code embed.FS
