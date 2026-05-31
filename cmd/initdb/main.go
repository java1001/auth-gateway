package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/auth-gateway/config"
	"github.com/auth-gateway/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	var siteHost string
	flag.StringVar(&siteHost, "db", "", "initialize only one site database by host, e.g. site3.com; leave empty to initialize all configured sites")
	flag.Parse()

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	targetSites, err := selectSites(cfg, siteHost)
	if err != nil {
		log.Fatalf("site selection failed: %v", err)
	}

	if err := database.InitAllDatabases(targetSites); err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}

	if siteHost != "" {
		fmt.Printf("Databases initialized for %s\n", siteHost)
		return
	}

	fmt.Println("Databases initialized")
}

func selectSites(cfg *config.Config, siteHost string) (map[string]config.SiteDBConfig, error) {
	if siteHost == "" {
		return cfg.Sites, nil
	}

	siteHost = strings.ToLower(strings.TrimSpace(siteHost))
	siteCfg, ok := cfg.Sites[siteHost]
	if !ok {
		return nil, fmt.Errorf("site %q is not configured; available sites: %s", siteHost, strings.Join(sortedSites(cfg.Sites), ", "))
	}

	return map[string]config.SiteDBConfig{
		siteHost: siteCfg,
	}, nil
}

func sortedSites(sites map[string]config.SiteDBConfig) []string {
	hosts := make([]string, 0, len(sites))
	for host := range sites {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts
}
