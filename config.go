package main

import (
	"log"
	"os"
	"strconv"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		log.Fatalf("invalid %s=%q: must be an integer number of bytes", key, v)
	}
	return n
}

type Config struct {
	Addr        string
	SitesDir    string
	BaseDomain  string
	MaxUpload   int64 // bytes; max request size
	MaxUnzipped int64 // bytes; total after decompression (zip-bomb guard)
	MaxFiles    int64 // max number of entries in an uploaded archive
}

func LoadConfig() Config {
	return Config{
		Addr:        env("ADDR", ":8000"),
		SitesDir:    env("SITES_DIR", "/var/www/html"),
		BaseDomain:  env("BASE_DOMAIN", "example.com"),
		MaxUpload:   envInt64("MAX_UPLOAD", 50<<20),    // 50 MB request
		MaxUnzipped: envInt64("MAX_UNZIPPED", 200<<20), // 200 MB total after decompression
		MaxFiles:    envInt64("MAX_FILES", 10_000),
	}
}
