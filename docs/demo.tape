# portspy demo tape — renders the hero GIF used in the README.
#
# Requires VHS (https://github.com/charmbracelet/vhs) and Python 3 (only for the
# throwaway demo servers). Then, from the repo root:
#
#     go build -o /tmp/portspy ./cmd/portspy
#     PATH=/tmp:$PATH vhs docs/demo.tape
#
# The hidden block below spins up two recognizable demo servers so the GIF is
# self-contained. Swap in real dev servers (vite, next dev, postgres…) for an
# even flashier capture.

Output docs/demo.gif

Set Shell bash
Set FontSize 18
Set Width 1100
Set Height 640
Set Padding 18
Set Theme "Catppuccin Mocha"

# --- hidden setup: two labeled listeners (a "node" and a "go" project) --------
Hide
Type "mkdir -p /tmp/portspy-demo/storefront /tmp/portspy-demo/api" Enter
Type `echo '{"name":"storefront"}' > /tmp/portspy-demo/storefront/package.json` Enter
Type `printf 'module github.com/me/api\ngo 1.22\n' > /tmp/portspy-demo/api/go.mod` Enter
Type "(cd /tmp/portspy-demo/storefront && python3 -m http.server 3000 >/dev/null 2>&1 &)" Enter
Type "(cd /tmp/portspy-demo/api && python3 -m http.server 8080 >/dev/null 2>&1 &)" Enter
Sleep 800ms
Type "clear" Enter
Show

# --- the demo -----------------------------------------------------------------
Type "portspy" Sleep 600ms Enter
Sleep 2s

Down Sleep 700ms
Enter Sleep 2500ms      # open the detail view (executable, project root, lineage)
Escape Sleep 600ms

Type "/" Sleep 400ms    # filter…
Type "3000" Sleep 1200ms
Enter Sleep 800ms

Type "x" Sleep 1500ms   # kill, with confirmation…
Type "y" Sleep 2200ms   # …port freed
Sleep 1s
Type "q"

# --- hidden teardown ----------------------------------------------------------
Hide
Type "pkill -f 'http.server 8080' >/dev/null 2>&1; rm -rf /tmp/portspy-demo" Enter
Show
