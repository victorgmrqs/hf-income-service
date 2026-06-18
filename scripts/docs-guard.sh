#!/usr/bin/env bash
# docs-guard (hf-income-service) — rede de segurança determinística do CI.
# Falha o PR se o código mudou sem documentação/teste correspondente no mesmo diff.
# Uso: scripts/docs-guard.sh [base-ref]   (default: origin/development)
set -euo pipefail

BASE="${1:-origin/development}"
CHANGED=$(git diff --name-only "$BASE"...HEAD)
fail=0

has() { echo "$CHANGED" | grep -qE "$1"; }
err() { echo "❌ docs-guard: $1"; fail=1; }

# 1. Mudança em src/internal/ exige entrada no CHANGELOG.
if has '^src/internal/' && ! has '^CHANGELOG\.md$'; then
  err "código em src/internal/ mudou sem entrada no CHANGELOG.md ([Unreleased])."
fi

# 2. Mudança de dependências exige CHANGELOG (seção Dependencies).
if has '^go\.mod$' && ! has '^CHANGELOG\.md$'; then
  err "go.mod mudou sem entrada no CHANGELOG.md (Dependencies)."
fi

# 3. Código de produção em src/internal/ exige um teste (*_test.go) no mesmo diff.
PROD=$(echo "$CHANGED" | grep -E '^src/internal/.*\.go$' | grep -vE '_test\.go$' || true)
if [ -n "$PROD" ] && ! echo "$CHANGED" | grep -qE '_test\.go$'; then
  err "código em src/internal/ mudou sem nenhum teste (*_test.go) no mesmo diff."
fi

# 4. Mudança de contrato REST exige openapi.yaml.
if has '^src/internal/handler/' && ! has '^openapi\.ya?ml$'; then
  echo "⚠️  docs-guard: handler alterado — confirme se openapi.yaml precisa de atualização."
fi

if [ "$fail" -eq 1 ]; then
  echo ""
  echo "Atualize a documentação/teste no mesmo branch e faça novo push."
  exit 1
fi
echo "✅ docs-guard: documentação consistente com o diff."
