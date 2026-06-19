#!/usr/bin/env bash
#
# publish.sh — fresh upload of portspy to a brand-new GitHub repo.
#
# Use this AFTER deleting the old repo, for absolute certainty that no old
# commit objects survive anywhere:
#
#     gh auth refresh -s delete_repo            # one-time, if needed
#     gh repo delete agenticraptor/portspy --yes
#     cd ~/Desktop/oss_mission/projects/portspy
#     bash publish.sh
#
# It builds a clean history from the CURRENT files (authored by your GitHub
# handle with a no-reply email), creates a fresh PRIVATE repo, and pushes —
# no force-push, because the repo starts empty. Uses your existing `gh` login.
#
set -euo pipefail

REPO_NAME="portspy"
DESC="🔌 See exactly what's running on every local port — and free it up in a single keystroke. A project-aware Go binary (TUI + CLI)."
TOPICS="cli,tui,golang,go,developer-tools,devtools,terminal,ports,localhost,kill-port,lsof,netstat,process,productivity,bubbletea,charm"

cd "$(dirname "$0")"

# --- Guard: no personal email may appear in any tracked file ------------------
# Generic check (no personal data is baked into this script): flag any email
# address that isn't an allow-listed no-reply/bot placeholder. The publish
# scripts and go.sum are excluded (scripts are never committed; go.sum has only
# module hashes).
EMAIL_RE='[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'
ALLOW_RE='users\.noreply\.github\.com|@portspy\.dev'

echo "==> Guard: scanning working tree for personal email addresses"
leak=$(grep -rInE --exclude-dir=.git --exclude=publish.sh --exclude=push.sh --exclude=go.sum -- "$EMAIL_RE" . 2>/dev/null | grep -vE "$ALLOW_RE" || true)
if [ -n "$leak" ]; then
  echo "    ✗ unexpected email address(es):"; echo "$leak" | sed 's/^/        /'
  echo "Aborting: scrub these before publishing."; exit 1
fi
echo "    ✓ clean"

# --- Auth ---------------------------------------------------------------------
echo "==> Checking gh authentication"
gh auth status >/dev/null 2>&1 || { echo "Not logged in. Run: gh auth login"; exit 1; }
OWNER=$(gh api user --jq .login)
UID_NUM=$(gh api user --jq .id)
NOREPLY="${UID_NUM}+${OWNER}@users.noreply.github.com"
echo "    authenticated as $OWNER (commits authored as $NOREPLY)"

# --- Refuse to clobber an existing repo (this script is for a fresh upload) ---
if gh repo view "$OWNER/$REPO_NAME" >/dev/null 2>&1; then
  echo "Repo $OWNER/$REPO_NAME already exists."
  echo "For a guaranteed-clean fresh upload, delete it first:"
  echo "    gh auth refresh -s delete_repo   # one-time, if needed"
  echo "    gh repo delete $OWNER/$REPO_NAME --yes"
  echo "(or use push.sh to force-update the existing repo in place)."
  exit 1
fi

# --- Rebuild a clean, single history from the current files -------------------
echo "==> Building clean history"
rm -rf .git
git init -q -b main
git config user.name "$OWNER"
git config user.email "$NOREPLY"
git config commit.gpgsign false

# Clean, conventional commits — no AI/co-author trailers. The publish scripts
# are intentionally NOT added, so they never reach the repo.
git add go.mod go.sum .gitignore .editorconfig Makefile .golangci.yml .goreleaser.yaml \
        cmd internal ':(exclude)*_test.go'
git commit -q -m "feat: implement portspy core (ports engine, killer, render, TUI, CLI)"
git add '*_test.go'
git commit -q -m "test: add unit test suite across packages"
git add README.md LICENSE CHANGELOG.md CONTRIBUTING.md CODE_OF_CONDUCT.md SECURITY.md docs
git commit -q -m "docs: add README, usage/platform guides, and community health files"
git add .github
git commit -q -m "ci: add CI, release (GoReleaser), issue/PR templates, and dependabot"
git tag -a v0.1.0 -m "portspy v0.1.0"

# --- Guard: the committed history itself must be clean ------------------------
echo "==> Guard: scanning committed history for personal email addresses"
hleak=$(git grep -InE -- "$EMAIL_RE" "$(git rev-list --all)" 2>/dev/null | grep -vE "$ALLOW_RE" || true)
if [ -n "$hleak" ]; then
  echo "    ✗ unexpected email address(es) in committed history:"; echo "$hleak" | sed 's/^/        /'
  exit 1
fi
echo "    ✓ history clean"

# --- Create the fresh repo and push ------------------------------------------
echo "==> Configuring git to use your gh credentials"
gh auth setup-git

echo "==> Creating private repo $OWNER/$REPO_NAME"
gh repo create "$OWNER/$REPO_NAME" --private --description "$DESC" >/dev/null

git remote remove origin 2>/dev/null || true
git remote add origin "https://github.com/$OWNER/$REPO_NAME.git"

push_ref () {  # $1 = ref
  if git push -u origin "$1"; then return 0; fi
  echo "    (helper push failed; retrying with gh token)"
  git push "https://x-access-token:$(gh auth token)@github.com/$OWNER/$REPO_NAME.git" "$1"
}

echo "==> Pushing main"
push_ref main
echo "==> Pushing v0.1.0 tag (triggers the GoReleaser release workflow)"
push_ref v0.1.0

echo "==> Setting topics"
gh repo edit "$OWNER/$REPO_NAME" --add-topic "$TOPICS" || true

URL="https://github.com/$OWNER/$REPO_NAME"
echo
echo "✅ Fresh upload complete: $URL"
echo "   • Actions:  $URL/actions   (CI + release running)"
echo "   • Release:  $URL/releases  (v0.1.0 from the clean tree)"
echo
echo "When you're ready to launch publicly:"
echo "   gh repo edit $OWNER/$REPO_NAME --visibility public --accept-visibility-change-consequences"
echo
echo "You can delete the helper scripts when done:  rm push.sh publish.sh"
