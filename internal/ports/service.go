package ports

import "strings"

// serviceRule maps a substring of the (lower-cased) process name + command line
// to a friendly service label. The first matching rule wins, so more specific
// needles are listed first.
type serviceRule struct {
	needle string
	name   string
}

// serviceRules is a curated list of common dev servers, build tools, and
// infrastructure daemons. Needles are chosen to be specific enough to avoid
// false positives against ordinary words.
var serviceRules = []serviceRule{
	// JS/TS dev servers & bundlers.
	{"next-server", "Next.js"},
	{"next dev", "Next.js"},
	{"nuxt", "Nuxt"},
	{"react-scripts", "Create React App"},
	{"vue-cli-service", "Vue CLI"},
	{"ng serve", "Angular"},
	{"@angular", "Angular"},
	{"vite", "Vite"},
	{"webpack-dev-server", "webpack"},
	{"webpack", "webpack"},
	{"astro", "Astro"},
	{"remix", "Remix"},
	{"gatsby", "Gatsby"},
	{"storybook", "Storybook"},
	{"esbuild", "esbuild"},
	{"parcel", "Parcel"},
	{"turbo", "Turborepo"},
	{"http-server", "static server"},
	{"live-server", "static server"},
	{"deno", "Deno"},
	{"bun ", "Bun"},

	// Python / Ruby / PHP / JVM web frameworks.
	{"runserver", "Django"},
	{"django", "Django"},
	{"uvicorn", "Uvicorn"},
	{"gunicorn", "Gunicorn"},
	{"hypercorn", "Hypercorn"},
	{"flask", "Flask"},
	{"fastapi", "FastAPI"},
	{"puma", "Puma (Rails)"},
	{"rails", "Rails"},
	{"artisan", "Laravel"},
	{"php -s", "PHP"},
	{"spring", "Spring Boot"},
	{"jekyll", "Jekyll"},
	{"hugo", "Hugo"},

	// Databases, caches, queues, and infra.
	{"postgres", "PostgreSQL"},
	{"pg_ctl", "PostgreSQL"},
	{"mysqld", "MySQL"},
	{"mariadb", "MariaDB"},
	{"redis-server", "Redis"},
	{"mongod", "MongoDB"},
	{"memcached", "Memcached"},
	{"elasticsearch", "Elasticsearch"},
	{"opensearch", "OpenSearch"},
	{"clickhouse", "ClickHouse"},
	{"cockroach", "CockroachDB"},
	{"kafka", "Kafka"},
	{"rabbitmq", "RabbitMQ"},
	{"nats-server", "NATS"},

	// Reverse proxies, object stores, and platform daemons.
	{"docker-proxy", "Docker"},
	{"com.docker", "Docker"},
	{"nginx", "nginx"},
	{"caddy", "Caddy"},
	{"traefik", "Traefik"},
	{"minio", "MinIO"},
	{"ollama", "Ollama"},
	{"vault", "Vault"},
	{"consul", "Consul"},
	{"etcd", "etcd"},
	{"prometheus", "Prometheus"},
	{"grafana", "Grafana"},
	{"supabase", "Supabase"},
}

// detectService guesses a friendly service label from a process name and its
// command line. It returns "" when nothing recognizable matches.
func detectService(process, command string) string {
	hay := " " + strings.ToLower(process) + " " + strings.ToLower(command) + " "
	for _, r := range serviceRules {
		if strings.Contains(hay, r.needle) {
			return r.name
		}
	}
	return ""
}
