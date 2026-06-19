#!/usr/bin/env bash
#
# push.sh — (re)publish portspy with a clean, single-line commit history.
#
# Run this ON YOUR MAC from inside the project folder:
#
#     cd ~/Desktop/oss_mission/projects/portspy
#     bash push.sh
#
# What it does:
#   • Refuses to run if any personal data is still present in the tree.
#   • Rebuilds git history from the CURRENT files (one clean history, no old
#     commits), authored by your GitHub handle with a no-reply email.
#   • Deletes the previous v0.1.0 tag AND its GitHub release (which carried the
#     old history), then FORCE-PUSHES the rebuilt history and a fresh tag.
#
# ⚠️  This rewrites history and force-pushes. That is intentional here: it is how
#     we guarantee no old commit/tag retains the previous contents. Safe to
#     re-run. Uses your existing `gh` login (no token needed).
#
set -euo pipefail

REPO_NAME="portspy"
DESC="🔌 See exactly what's running on every local port — and free it up in a single keystroke. A project-aware Go binary (TUI + CLI)."
TOPICS="cli,tui,golang,go,developer-tools,devtools,terminal,ports,localhost,kill-port,lsof,netstat,process,productivity,bubbletea,charm"

cd "$(dirname "$0")"

# Any email address in a tracked file other than these allow-listed ones is
# treated as a leak. This keeps the guard generic — no personal data is baked
# into this script. (push.sh and go.sum are excluded; push.sh is never
# committed, go.sum holds only module hashes.)
EMAIL_RE='[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'
ALLOW_RE='users\.noreply\.github\.com|@portspy\.dev'

echo "==> Guard: scanning working tree for personal email addresses"
leak=$(grep -rInE --exclude-dir=.git --exclude=push.sh --exclude=go.sum -- "$EMAIL_RE" . 2>/dev/null | grep -vE "$ALLOW_RE" || true)
if [ -n "$leak" ]; then
  echo "    ✗ unexpected email address(es):"; echo "$leak" | sed 's/^/        /'
  echo "Aborting: scrub these before publishing."; exit 1
fi
echo "    ✓ clean"

echo "==> Checking gh authentication"
gh auth status >/dev/null 2>&1 || { echo "Not logged in. Run: gh auth login"; exit 1; }
OWNER=$(gh api user --jq .login)
UID_NUM=$(gh api user --jq .id)
NOREPLY="${UID_NUM}+${OWNER}@users.noreply.github.com"
echo "    authenticated as $OWNER (commits will be authored as $NOREPLY)"

echo "==> Rebuilding a clean history from the current files"
rm -rf .git
git init -q -b main
git config user.name "$OWNER"
git config user.email "$NOREPLY"
git config commit.gpgsign false

# Clean, conventional commits — no AI/co-author trailers.
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

# Final guard: make sure the committed history itself is clean.
echo "==> Guard: scanning committed history for personal email addresses"
hleak=$(git grep -InE -- "$EMAIL_RE" $(git rev-list --all) 2>/dev/null | grep -vE "$ALLOW_RE" || true)
if [ -n "$hleak" ]; then
  echo "    ✗ unexpected email address(es) in committed history:"; echo "$hleak" | sed 's/^/        /'
  exit 1
fi
echo "    ✓ history clean"

echo "==> Configuring git to use your gh credentials"
gh auth setup-git

if ! gh repo view "$OWNER/$REPO_NAME" >/dev/null 2>&1; then
  echo "==> Creating private repo $OWNER/$REPO_NAME"
  gh repo create "$OWNER/$REPO_NAME" --private --description "$DESC" >/dev/null
fi

TOKEN_URL="https://x-access-token:$(gh auth token)@github.com/$OWNER/$REPO_NAME.git"

echo "==> Deleting any previous v0.1.0 release and tag (they carried the old history)"
gh release delete v0.1.0 --repo "$OWNER/$REPO_NAME" --yes --cleanup-tag 2>/dev/null || true
git push "$TOKEN_URL" :refs/tags/v0.1.0 2>/dev/null || true   # belt-and-suspenders

git remote remove origin 2>/dev/null || true
git remote add origin "https://github.com/$OWNER/$REPO_NAME.git"

push_force () {  # $1 = ref
  if git push --force origin "$1"; then return 0; fi
  echo "    (helper push failed; retrying with gh token)"
  git push --force "$TOKEN_URL" "$1"
}

echo "==> Force-pushing the rebuilt main"
push_force main
git branch --set-upstream-to=origin/main main >/dev/null 2>&1 || true

echo "==> Pushing the fresh v0.1.0 tag (triggers a clean GoReleaser release)"
push_force v0.1.0

echo "==> Setting topics"
gh repo edit "$OWNER/$REPO_NAME" --add-topic "$TOPICS" || true

URL="https://github.com/$OWNER/$REPO_NAME"
echo
echo "✅ Done: $URL"
echo "   • Actions:  $URL/actions   (CI + release running)"
echo "   • Release:  $URL/releases  (fresh v0.1.0 from the clean tree)"
echo
echo "When you're ready to launch publicly:"
echo "   gh repo edit $OWNER/$REPO_NAME --visibility public --accept-visibility-change-consequences"
echo
echo "Note: GitHub may briefly retain now-unreachable old commits by SHA. For a"
echo "private, unforked repo this is harmless. For absolute certainty you can"
echo "delete and recreate the repo:"
echo "   gh repo delete $OWNER/$REPO_NAME --yes && bash push.sh"
