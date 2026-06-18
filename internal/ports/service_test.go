package ports

import "testing"

func TestDetectService(t *testing.T) {
	cases := []struct {
		process string
		command string
		want    string
	}{
		{"node", "node node_modules/.bin/vite", "Vite"},
		{"node", "next-server (v14)", "Next.js"},
		{"node", "node /app/node_modules/react-scripts/scripts/start.js", "Create React App"},
		{"postgres", "/usr/local/bin/postgres -D /data", "PostgreSQL"},
		{"redis-server", "redis-server *:6379", "Redis"},
		{"python3", "python -m uvicorn main:app", "Uvicorn"},
		{"ruby", "puma 6.0 (tcp://0.0.0.0:3000)", "Puma (Rails)"},
		{"docker-proxy", "/usr/bin/docker-proxy -container-port 80", "Docker"},
		{"ollama", "ollama serve", "Ollama"},
		{"node", "node server.js", ""}, // nothing recognizable
		{"mystery", "", ""},
	}
	for _, c := range cases {
		if got := detectService(c.process, c.command); got != c.want {
			t.Errorf("detectService(%q, %q) = %q, want %q", c.process, c.command, got, c.want)
		}
	}
}
