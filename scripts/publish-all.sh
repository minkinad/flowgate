#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
workspace_dir="$(cd -- "$script_dir/../.." && pwd)"
owner="minkinad"
repos=(flowgate portsentinel web-config-guard)

for command_name in git gh timeout; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf 'Нужна команда: %s\n' "$command_name" >&2
    exit 1
  fi
done

if ! gh auth token >/dev/null 2>&1; then
  printf 'GitHub CLI не авторизован. Выполните gh auth login и повторите запуск.\n' >&2
  exit 1
fi

if ! timeout 30s gh api user --jq .login >/dev/null; then
  printf 'Не удалось проверить GitHub API. Проверьте сеть и gh auth status; push ещё не выполнялся.\n' >&2
  exit 1
fi

for repo in "${repos[@]}"; do
  repo_dir="$workspace_dir/$repo"
  if [[ ! -d "$repo_dir/.git" ]]; then
    printf 'Не найден Git репозиторий: %s\n' "$repo_dir" >&2
    exit 1
  fi
  if [[ "$(git -C "$repo_dir" branch --show-current)" != main ]]; then
    printf 'Ожидалась ветка main: %s\n' "$repo_dir" >&2
    exit 1
  fi
  if [[ -n "$(git -C "$repo_dir" status --porcelain)" ]]; then
    printf 'Есть незакоммиченные изменения: %s\n' "$repo_dir" >&2
    exit 1
  fi
  if ! timeout 30s gh repo view "$owner/$repo" --json name --jq .name >/dev/null; then
    printf 'Нет доступа к репозиторию %s/%s; push ещё не выполнялся.\n' "$owner" "$repo" >&2
    exit 1
  fi
done

printf '\nПубликация веток и аннотированных тегов\n'
for repo in "${repos[@]}"; do
  printf '\n== %s ==\n' "$repo"
  git -C "$workspace_dir/$repo" push -u origin main --follow-tags
done

printf '\nПубликация подготовленных issues\n'
created=0
skipped=0
for repo in "${repos[@]}"; do
  drafts_dir="$workspace_dir/$repo/.github/issue-drafts"
  existing_titles="$(timeout 30s gh issue list --repo "$owner/$repo" --state all --limit 500 --json title --jq '.[].title')"
  for draft in "$drafts_dir"/*.md; do
    [[ -f "$draft" ]] || continue
    title="$(head -n 1 "$draft")"
    title="${title#\# }"
    if printf '%s\n' "$existing_titles" | grep -Fxq -- "$title"; then
      printf 'Уже существует: %s — %s\n' "$repo" "$title"
      skipped=$((skipped + 1))
      continue
    fi
    gh issue create --repo "$owner/$repo" --title "$title" --body-file "$draft"
    created=$((created + 1))
  done
done
printf '\nГотово: создано issues: %d; уже существовали: %d.\n' "$created" "$skipped"
