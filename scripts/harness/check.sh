#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

# Mantiene la verificacion autocontenida en sandboxes y evita escribir en la
# cache global del usuario. Un GOCACHE explicito del entorno conserva prioridad.
if [ -z "${GOCACHE:-}" ]; then
  export GOCACHE="${TMPDIR:-/tmp}/repository-go-build"
fi

log() {
  printf '==> %s\n' "$1"
}

die() {
  printf 'ERROR: %s\n' "$1" >&2
  exit 1
}

require_file() {
  [ -f "$1" ] || die "falta $1; restaura la fuente de verdad requerida"
}

log "validando estructura del harness"

required_files=(
  AGENTS.md
  ARCHITECTURE.md
  CHECKPOINTS.md
  README.md
  .codex/config.toml
  .codex/agents/leader.toml
  .codex/agents/explorer.toml
  .codex/agents/implementer.toml
  .codex/agents/reviewer.toml
  docs/README.md
  docs/PRODUCT.md
  docs/ENGINEERING.md
  docs/VERIFICATION.md
  docs/AGENT_WORKFLOWS.md
  docs/exec-plans/TEMPLATE.md
  api/openapi/v1/openapi.yaml
  go.mod
)

for required_file in "${required_files[@]}"; do
  require_file "$required_file"
done

[ -d docs/exec-plans/active ] || die "falta docs/exec-plans/active"
[ -d docs/exec-plans/completed ] || die "falta docs/exec-plans/completed"

grep -Eq '^\[agents\]$' .codex/config.toml || die ".codex/config.toml debe declarar [agents]"
grep -Eq '^enabled = true$' .codex/config.toml || die "multiagente debe estar habilitado en .codex/config.toml"

for agent_file in .codex/agents/*.toml; do
  grep -Eq '^name = "[a-z][a-z0-9_]*"$' "$agent_file" || die "$agent_file necesita un name valido"
  grep -Eq '^description = ".+"$' "$agent_file" || die "$agent_file necesita description"
  grep -Eq '^developer_instructions = """$' "$agent_file" || die "$agent_file necesita developer_instructions"
  agent_name="$(sed -n 's/^name = "\([^"]*\)"$/\1/p' "$agent_file")"
  agent_stem="$(basename "$agent_file" .toml)"
  [ "$agent_name" = "$agent_stem" ] || die "$agent_file debe llamarse $agent_name.toml"
done

agents_lines="$(wc -l < AGENTS.md | tr -d ' ')"
[ "$agents_lines" -le 120 ] || die "AGENTS.md tiene $agents_lines lineas; mantenlo como mapa (maximo 120)"

for adr_file in docs/adr/[0-9][0-9][0-9][0-9]-*.md; do
  [ -e "$adr_file" ] || continue
  adr_name="$(basename "$adr_file")"
  grep -Fq "$adr_name" docs/README.md || die "$adr_name no aparece en docs/README.md"
done

log "validando limites arquitectonicos"

module_path="$(sed -n 's/^module[[:space:]][[:space:]]*//p' go.mod)"
[ -n "$module_path" ] || die "go.mod debe declarar el module path"

for generic_name in common utils models interfaces helpers; do
  generic_dir="$(find internal -type d -name "$generic_name" -print -quit)"
  [ -z "$generic_dir" ] || die "paquete generico prohibido: $generic_dir; usa una responsabilidad concreta"
done

for domain_dir in internal/ticket internal/draw; do
  while IFS= read -r domain_file; do
    for infrastructure_package in server mongodb config; do
      if grep -Fn "\"$module_path/internal/$infrastructure_package" "$domain_file"; then
        die "$domain_file hace depender el negocio de infraestructura"
      fi
    done
  done < <(find "$domain_dir" -type f -name '*.go' -print)
done

grep -Eq '^openapi: 3\.' api/openapi/v1/openapi.yaml || die "OpenAPI v1 debe declarar una especificacion 3.x"
grep -Eq '^paths:' api/openapi/v1/openapi.yaml || die "OpenAPI v1 debe declarar paths"
grep -Eq '^components:' api/openapi/v1/openapi.yaml || die "OpenAPI v1 debe declarar components"

log "validando toolchain Go"

command -v go >/dev/null 2>&1 || die "Go no esta disponible en PATH"
command -v gofmt >/dev/null 2>&1 || die "gofmt no esta disponible en PATH"

go mod tidy -diff

go_packages="$(go list ./... 2>/dev/null || true)"
if [ -n "$go_packages" ]; then
  unformatted="$(find cmd internal -type f -name '*.go' -exec gofmt -l {} +)"
  [ -z "$unformatted" ] || die "archivos sin gofmt:\n$unformatted"

  go vet ./...
  go test ./...
else
  printf '%s\n' "SKIP: aun no existen paquetes Go compilables"
fi

log "harness verde"
